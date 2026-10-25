# Prise en main

Ce guide construit une application en cinq étapes, chacune lançable. Comptez une vingtaine de minutes.

À la fin, vous aurez un assistant avec un agent qui répond en streaming, une commande `/` et une page à vous, le tout en français et en anglais.

## Avant de commencer

- Go 1.26 ou plus récent (`go version`).
- Sous Linux ou macOS, les [prérequis système](../README.md#plateformes) de Gio.

## Étape 1 : créer le projet

```bash
mkdir mon-assistant
cd mon-assistant
go mod init mon-assistant
go get github.com/sitle/ori-agent@latest
```

Créez `main.go` :

```go
package main

import oriagent "github.com/sitle/ori-agent"

func main() {
	oriagent.Run(oriagent.Config{AppName: "Mon assistant"})
}
```

```bash
go run .
```

La fenêtre s'ouvre avec toute l'interface. Si vous envoyez un message, une réponse vous indique qu'aucun agent n'est configuré : c'est l'étape suivante.

## Étape 2 : écrire un agent

Un agent est une fonction appelée à chaque message envoyé. Elle reçoit un `Turn`, qui contient la demande et permet d'écrire la réponse.

```go
package main

import (
	"context"
	"strings"
	"time"

	oriagent "github.com/sitle/ori-agent"
)

func main() {
	oriagent.Run(oriagent.Config{
		AppName: "Mon assistant",
		Agent:   oriagent.AgentFunc(repondre),
	})
}

func repondre(ctx context.Context, t *oriagent.Turn) error {
	b := t.Text()
	defer b.Done()
	for _, mot := range strings.Fields("Bonjour ! Vous avez écrit : " + t.Prompt) {
		select {
		case <-ctx.Done(): // l'utilisateur a cliqué sur Stop
			return ctx.Err()
		case <-time.After(60 * time.Millisecond):
		}
		b.Write(mot + " ")
	}
	return nil
}
```

Le texte s'affiche mot à mot, et le bouton Envoyer devient Arrêter pendant la réponse. Le guide [Écrire un agent](agents.md) montre comment afficher une réflexion, des appels d'outils, des tâches et des sous-agents, et comment brancher un vrai LLM.

## Étape 3 : ajouter une commande

Les commandes s'ouvrent en tapant `/` dans la zone de saisie. Ajoutez la vôtre à côté des commandes intégrées :

```go
var bonjour = &oriagent.Command{
	Name: "bonjour",
	Args: "[prénom]",
	Desc: "Dire bonjour",
	Run: func(c *oriagent.Ctl, args string) string {
		if args == "" {
			args = "tout le monde"
		}
		return "Bonjour, " + args + " !" // affiché dans le fil
	},
}
```

```go
oriagent.Run(oriagent.Config{
	AppName:  "Mon assistant",
	Agent:    oriagent.AgentFunc(repondre),
	Commands: []*oriagent.Command{bonjour},
})
```

Tapez `/bon` : la palette filtre et propose `/bonjour`. Le paramètre `c` donne accès à l'application : session en cours, notifications, aperçu, langue, lancement d'un tour d'agent… (voir [Personnaliser l'application](personnalisation.md#commandes)).

## Étape 4 : ajouter une page

Une page apparaît dans la barre latérale. Vous dessinez son contenu avec le kit `ui` ; ori-agent ajoute le titre et la colonne qui défile.

```go
import (
	"gioui.org/font"
	"gioui.org/layout"

	"github.com/sitle/ori-agent/i18n"
	"github.com/sitle/ori-agent/ui"
)

var aPropos = oriagent.Page{
	ID:    "a-propos",
	Title: "À propos",
	Layout: func(gtx layout.Context, th *ui.Theme, tr i18n.Translator) layout.Dimensions {
		return th.Section(gtx, "Mon assistant", "Version 1.0",
			th.Label(14, "Un assistant construit avec ori-agent.", th.Text, font.Normal).Layout)
	},
}
```

```go
Pages: []oriagent.Page{aPropos},
```

## Étape 5 : traduire

Remplacez les textes par des clés et fournissez les traductions :

```go
var textes = map[string]map[string]string{
	"fr": {"bonjour.desc": "Dire bonjour", "apropos.titre": "À propos"},
	"en": {"bonjour.desc": "Say hello", "apropos.titre": "About"},
}
```

```go
bonjour.Desc = "bonjour.desc"
aPropos.Title = "apropos.titre"

oriagent.Run(oriagent.Config{
	// …
	Translations: textes,
})
```

Lancez l'application, puis tapez `/lang en` : toute l'interface passe en anglais, y compris vos textes. Dans une page, utilisez `tr.T("clé")` ; dans une commande, `c.T("clé")`. Le guide [Langues de l'interface](langues.md) explique le fonctionnement complet.

## Et ensuite ?

- Ajouter des données de démonstration (projets, sessions, transcripts) : [Personnaliser l'application](personnalisation.md#données-de-démonstration).
- Désactiver ce dont vous n'avez pas besoin (voix, Pomodoro, profil…) : [Personnaliser l'application](personnalisation.md#fonctionnalités).
- Lire le code de l'exemple complet : [`examples/anna`](../examples/anna/main.go).
