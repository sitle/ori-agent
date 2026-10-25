package oriagent

import (
	"context"
	"errors"
	"testing"
	"time"
)

var testUser = User{First: "Hina", Last: "Teriitahi", Email: "hina@example.pf", Service: "DSI", Role: "Dev"}

// locked runs f with the state locked, as the frame does.
func locked(a *App, f func()) {
	a.mu.Lock()
	defer a.mu.Unlock()
	f()
}

// waitFor polls cond with the state locked.
func waitFor(t *testing.T, a *App, cond func() bool) {
	t.Helper()
	for end := time.Now().Add(2 * time.Second); time.Now().Before(end); time.Sleep(5 * time.Millisecond) {
		ok := false
		locked(a, func() { ok = cond() })
		if ok {
			return
		}
	}
	t.Fatal("timed out")
}

func TestSaveProfileInMemory(t *testing.T) {
	a := testApp(Config{User: testUser})
	a.profile.fields[pRole].SetText("  Lead ")
	a.saveProfile()
	if u := a.user(); u.Role != "Lead" || u.First != "Hina" {
		t.Errorf("user = %+v", u)
	}
}

func TestSaveProfileAsync(t *testing.T) {
	release := make(chan struct{})
	var got User
	a := testApp(Config{User: testUser, OnProfile: func(_ context.Context, u User) (User, error) {
		<-release
		got = u
		u.Email = testUser.Email // the application does not allow changing it
		return u, nil
	}})
	locked(a, func() {
		a.profile.fields[pFirst].SetText("Moana")
		a.profile.fields[pEmail].SetText("other@example.pf")
		a.saveProfile()
		if !a.profile.saving {
			t.Error("not saving while OnProfile runs")
		}
	})
	close(release)
	waitFor(t, a, func() bool { return !a.profile.saving })
	if got.First != "Moana" || got.Email != "other@example.pf" {
		t.Errorf("OnProfile got %+v", got)
	}
	want := testUser
	want.First = "Moana"
	if u := a.user(); u != want {
		t.Errorf("user = %+v, want %+v", u, want)
	}
	if e := a.profile.fields[pEmail].Text(); e != testUser.Email {
		t.Errorf("email field = %q", e)
	}
}

func TestSaveProfileError(t *testing.T) {
	a := testApp(Config{User: testUser, OnProfile: func(context.Context, User) (User, error) {
		return User{}, errors.New("offline")
	}})
	locked(a, func() {
		a.profile.fields[pFirst].SetText("Moana")
		a.saveProfile()
	})
	waitFor(t, a, func() bool { return !a.profile.saving })
	if u := a.user(); u != testUser {
		t.Errorf("saved user changed: %+v", u)
	}
	if f := a.profile.fields[pFirst].Text(); f != "Moana" {
		t.Errorf("edit lost: %q", f)
	}
}

func TestLogout(t *testing.T) {
	for _, fail := range []bool{false, true} {
		called := make(chan struct{}, 1)
		a := testApp(Config{OnLogout: func(context.Context) error {
			called <- struct{}{}
			if fail {
				return errors.New("offline")
			}
			return nil
		}})
		locked(a, a.logout)
		<-called
		waitFor(t, a, func() bool { return !a.profile.loggingOut })
	}
}

func TestProfileLock(t *testing.T) {
	a := testApp(Config{User: testUser, ProfileLock: ProfileLock{Email: true}})
	if !a.profile.fields[pEmail].ReadOnly || a.profile.fields[pFirst].ReadOnly {
		t.Error("only the email field should be read-only")
	}
}

func TestSetModels(t *testing.T) {
	var calls []Settings
	a := testApp(Config{Models: testModels(), WorkDir: t.TempDir(),
		OnSettings: func(s Settings) { calls = append(calls, s) }})
	c := &Ctl{a}
	c.SetModel(1) // Smart
	a.checkSettings()

	c.SetModels([]Model{{Name: "Tiny"}, {Name: "Smart"}})
	if c.Model() != 1 || len(a.setup.models) != 2 {
		t.Errorf("model = %d, %d buttons", c.Model(), len(a.setup.models))
	}
	a.checkSettings()
	if len(calls) != 1 {
		t.Errorf("OnSettings fired without a selection change: %d calls", len(calls))
	}

	c.SetModels([]Model{{Name: "Tiny"}, {Name: "Huge"}})
	a.checkSettings()
	if c.Model() != 0 || len(calls) != 2 || calls[1].Model != "Tiny" {
		t.Errorf("model = %d, calls = %+v", c.Model(), calls)
	}
}

func TestTurnSetUserAndModels(t *testing.T) {
	a := testApp(Config{Models: testModels(), WorkDir: t.TempDir()})
	done := make(chan struct{})
	locked(a, func() {
		a.startTurn(a.cur, "/login", nil, func(ctx context.Context, t *Turn) error {
			t.SetUser(testUser)
			t.SetModels([]Model{{Name: "Signed-in"}})
			close(done)
			<-ctx.Done() // never canceled: the end of a turn needs a window
			return nil
		})
	})
	<-done
	locked(a, func() {
		if u := a.user(); u != testUser {
			t.Errorf("user = %+v", u)
		}
		if a.fullName() != "Hina Teriitahi" {
			t.Errorf("sidebar name = %q", a.fullName())
		}
		if m := a.model().Name; m != "Signed-in" {
			t.Errorf("model = %q", m)
		}
	})
}
