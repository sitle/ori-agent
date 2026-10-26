# Personnaliser l'application

Tout se règle dans la structure `oriagent.Config` passée à `oriagent.Run`. Tous les champs sont facultatifs : une `Config` vide donne déjà une application complète.

## Identité et apparence

| Champ | Rôle | Par défaut |
|---|---|---|
| `AppName` | Nom affiché dans la barre de titre et l'écran d'accueil | `"Assistant"` |
| `WindowTitle` | Titre de la fenêtre pour le système | `AppName` |
| `Logo` | Logo SVG (`[]byte`, souvent via `//go:embed`) | un rond de couleur |
| `Light`, `Dark` | Palettes de couleurs (voir [Composants et thèmes](composants.md)) | Ori |
| `Fonts` | Polices (`[]font.FontFace`) | polices Go intégrées |
| `LightMode` | Démarrer en thème clair | thème sombre |
| `Width`, `Height` | Taille initiale de la fenêtre | 1280 × 820 |
| `Placeholder` | Texte d'invite de la zone de saisie | « Décrivez une tâche… » |
| `ModeLabel` | Badge de la zone de saisie (`"-"` le masque) | « Mode auto » |
| `EmptyTitle`, `EmptySubtitle` | Écran d'une session vide | « Sur quoi travaille-t-on ? », dossier de travail |

Les textes peuvent être des clés de traduction (voir [Langues](langues.md)).

## Agent, modèles et dossier

| Champ | Rôle |
|---|---|
| `Agent` | Le moteur qui répond (voir [Écrire un agent](agents.md)) |
| `Models` | Les modèles proposés dans la Configuration, par `/model` et dans la barre de titre. `Desc` peut être une clé de traduction |
| `WorkDir` | Dossier de travail ; par défaut, le dossier courant. L'utilisateur peut le changer dans la Configuration |

## Mémoriser les réglages

ori-agent n'enregistre rien lui-même. La structure `oriagent.Settings` réunit toutes les valeurs de la page Configuration, et l'application les conserve où elle veut :

| Champ | Valeur |
|---|---|
| `WorkDir` | Dossier de travail |
| `AllowCommands`, `AskEdits` | Autoriser les commandes, demander avant de modifier un fichier |
| `Thinking` | Réflexion étendue |
| `Notifications` | Notifications |
| `Model` | Le `Name` du modèle choisi |
| `Dark` | Thème sombre |
| `Locale` | Langue de l'interface |

- `Config.Settings` donne les valeurs de départ. Il prime sur `WorkDir`, `LightMode` et `Locale` ; un dossier qui n'existe plus, une langue sans catalogue ou un modèle inconnu sont ignorés. Sans lui, ce sont les valeurs de `oriagent.DefaultSettings()` : commandes, confirmation des modifications, réflexion et thème sombre activés.
- `Config.OnSettings` reçoit les nouvelles valeurs à chaque changement : interrupteurs et choix de la Configuration, bouton de thème de la barre de titre, `/model`, `/theme`, `/lang`, dossier appliqué par « Terminer ». Il est appelé à la fin de l'image, dans le fil de l'interface, état verrouillé : il doit rendre la main vite, donc écrire le fichier dans une goroutine.
- Une commande lit les valeurs courantes avec `c.Settings()` ; un agent avec `t.Thinking`, `t.AllowCommands`, `t.AskEdits`, `t.Model`, `t.WorkDir` et `t.Locale`.

`Settings` porte des étiquettes JSON :

```go
set := oriagent.DefaultSettings()
if data, err := os.ReadFile(chemin); err == nil {
	_ = json.Unmarshal(data, &set) // les champs absents gardent leur valeur par défaut
}
oriagent.Run(oriagent.Config{
	Settings: &set,
	OnSettings: func(s oriagent.Settings) {
		data, _ := json.MarshalIndent(s, "", "  ")
		go os.WriteFile(chemin, data, 0o600)
	},
})
```

## Commandes

Chaque commande est un `*oriagent.Command` :

```go
&oriagent.Command{
	Name: "export",          // tapée /export
	Args: "<fichier>",       // aide affichée dans la palette
	Desc: "Exporter la conversation",
	Run:  func(c *oriagent.Ctl, args string) string { /* … */ return "Exporté." },
}
```

- `Args` entre chevrons (`<x>`) indique un argument obligatoire : choisir la commande dans la palette la complète au lieu de l'exécuter. Entre crochets (`[x]`), l'argument est facultatif.
- Le texte renvoyé par `Run` s'affiche dans le fil ; une chaîne vide n'affiche rien.
- `Run` s'exécute dans le fil de l'interface : les traitements longs passent par `c.RunTurn` ou par une goroutine.

Ce que `Ctl` permet :

| Méthode | Effet |
|---|---|
| `Session()`, `Sessions()`, `Projects()` | Accéder aux sessions |
| `NewSession()`, `Clear()` | Nouvelle session, effacer la conversation |
| `RunTurn(prompt, fn)` | Lancer un tour d'agent (`fn` nil : l'agent configuré) |
| `Open(chemin)` | Ouvrir un fichier dans l'aperçu |
| `PickFiles()` | Ouvrir le sélecteur de pièces jointes |
| `Notify(msg)` | Notification passagère |
| `Show(vue)` | Afficher `"chat"`, `"setup"`, `"profile"` ou l'`ID` d'une page |
| `Models()`, `Model()`, `SetModel(i)`, `SetModels(ms)` | Modèle actif, liste des modèles |
| `User()`, `SetUser(u)` | Profil (voir [Profil](#profil)) |
| `Dark()`, `SetDark(b)` | Thème |
| `Settings()` | Valeurs de la page Configuration (voir [Mémoriser les réglages](#mémoriser-les-réglages)) |
| `Locale()`, `Locales()`, `SetLocale(l)`, `LanguageName(l)`, `T(clé)` | Langue et traductions |
| `ToggleSidebar()`, `ToggleVoice()`, `Pomodoro()` | Barre latérale, dictée, minuteur |
| `WorkDir()`, `Commands()` | Dossier de travail, liste des commandes |

Commandes intégrées : `/help`, `/new`, `/clear`, `/compact`, `/model`, `/open`, `/usage`, `/status`, `/lang`, `/sidebar`, `/upload`, `/voice`, `/pomodoro`, `/theme`, `/config`, `/profile`. Elles disparaissent avec `Disable.BuiltinCommands`, ou individuellement quand la fonctionnalité correspondante est désactivée.

## Pages

```go
oriagent.Page{
	ID:     "stats",                   // utilisé par Ctl.Show et StartView
	Title:  "Statistiques",            // ou une clé de traduction
	Icon:   monIcone,                  // facultatif : func(gtx, couleur) layout.Dimensions
	Layout: func(gtx layout.Context, th *ui.Theme, tr i18n.Translator) layout.Dimensions {
		return th.Section(gtx, tr.T("stats.titre"), "", contenu)
	},
}
```

Les pages s'affichent dans la barre latérale, au-dessus du profil. Le kit [`ui`](composants.md) fournit sections, panneaux, champs, interrupteurs, boutons, cartes de choix… L'exemple `examples/anna` contient une page « Raccourcis » complète.

## Profil

| Champ | Rôle |
|---|---|
| `User` | Prénom, nom, e-mail, service et fonction (`First`, `Last`, `Email`, `Service`, `Role`) |
| `Devices` | Appareils connectés (`Name`, `Detail`, `Current`) ; vide, la section est masquée |
| `ProfileLock` | Champs en lecture seule (`First`, `Last`, `Email`, `Service`, `Role`), par exemple ceux qui viennent d'un annuaire |
| `OnProfile` | Appelé avec les champs modifiés quand l'utilisateur clique sur « Enregistrer » |
| `OnLogout` | Appelé par le bouton « Se déconnecter » |

Sans `OnProfile`, les modifications restent en mémoire, et sans `OnLogout` la déconnexion est simulée. Les deux fonctions s'exécutent dans une goroutine : pendant ce temps les boutons de la page sont inactifs et affichent « Enregistrement… » ou « Déconnexion… ». Le `User` renvoyé par `OnProfile` remplace les champs et le pied de la barre latérale ; l'application peut le corriger, par exemple remettre une adresse qu'elle ne laisse pas modifier. Une erreur s'affiche dans une notification et les modifications restent dans les champs.

```go
oriagent.Config{
	User:        utilisateur,
	ProfileLock: oriagent.ProfileLock{Email: true},
	OnProfile: func(ctx context.Context, u oriagent.User) (oriagent.User, error) {
		return annuaire.Enregistrer(ctx, u) // renvoie le profil tel qu'enregistré
	},
	OnLogout: func(ctx context.Context) error { return session.Fermer(ctx) },
}
```

Le profil et les modèles fixés au démarrage peuvent changer ensuite, par exemple après une connexion : `c.SetUser(u)` et `c.SetModels(ms)` dans une commande, `t.SetUser(u)` et `t.SetModels(ms)` depuis un tour d'agent (lancé par `c.RunTurn`). Le modèle choisi reste sélectionné si son nom figure encore dans la liste, sinon c'est le premier ; `OnSettings` est appelé si la sélection change.

## Projets

Un projet regroupe des sessions dans la barre latérale. En plus de son nom et de ses sessions, il peut avoir :

| Champ | Rôle |
|---|---|
| `Description` | Un texte libre, modifiable par l'utilisateur (ce n'est pas une clé de traduction) |
| `WorkDir` | Son dossier : il devient `Turn.WorkDir` pour ses sessions et leurs sous-agents, et c'est lui qui est affiché, copié et proposé pour joindre un fichier. Vide : le dossier de la Configuration |
| `Permissions` | Ses autorisations (`AllowCommands`, `AskEdits`), transmises à l'agent à la place de celles de la Configuration. Nil : celles de la Configuration |
| `Permanent` | Pas de bouton de suppression sur sa page (par exemple le dossier où l'application a été lancée) |

Chaque ligne de projet porte un « + » qui démarre une session dedans, ou reprend sa session vide ; `c.NewSessionIn(p)` fait de même depuis une commande.

Trois rappels laissent l'application gérer les projets :

- **`OnNewProject(ctx, nom, description)`** : le bouton principal de la barre latérale devient « Nouveau projet » et ouvre un formulaire. L'application crée ce qui porte le projet, souvent un dossier, et renvoie le `*Project`.
- **`OnUpdateProject(ctx, projet, modif)`** : une roue dentée sur chaque ligne de projet ouvre sa page (nom, description, autorisations, dossier copiable). L'application enregistre la modification et la renvoie telle qu'elle l'a gardée.
- **`OnDeleteProject(ctx, projet)`** : ajoute à cette page un bouton de suppression, à confirmer, sauf pour un projet `Permanent`.

Ils s'exécutent dans une goroutine ; une erreur reste affichée dans le formulaire ou la page.

**`ProjectFiles(projet)`** ajoute à la page d'un projet une section « Fichiers » : l'application renvoie une liste de `ProjectFile` (`Name`, `Detail` pour la taille, la date ou une empreinte, `Path`, `Warning` affiché en rouge). Un clic sur un fichier dont `Path` est renseigné l'ouvre dans l'aperçu, à côté de la page. La liste est demandée dans une goroutine à chaque ouverture de la page ; un message d'attente s'affiche en attendant.

![Page d'un projet avec ses fichiers et l'aperçu de l'un d'eux](images/page-projet.png)

Pour tous ces rappels, l'exemple [`examples/anna/store.go`](../examples/anna/store.go) en donne une version complète : un dossier par projet, marqué par un fichier `project.json`, rechargé au démarrage, et une suppression qui ne touche jamais un dossier que l'application n'a pas créé.

## Données de démonstration

`Projects` remplit l'arborescence au démarrage. Des constructeurs facilitent l'écriture des transcripts :

```go
s := &oriagent.Session{Title: "Corriger la connexion", Messages: []*oriagent.Message{
	oriagent.UserMsg("Le bouton de connexion ne fait rien."),
	oriagent.ThinkingMsg("Le gestionnaire de clic n'est sans doute pas branché.", 3),
	oriagent.TasksMsg(oriagent.Task{Text: "Lire le code", Status: oriagent.TaskDone}),
	oriagent.ToolMsg("Read", "login.go", "   1  package main"),
	oriagent.TextMsg("Corrigé : le gestionnaire est maintenant enregistré."),
}}
s.Spawn("Lancer les tests") // un sous-agent

Projects: []*oriagent.Project{oriagent.NewProject("site-web", s)},
```

Mettez `Expanded: true` sur un message d'outil pour l'afficher déplié, `HasDiff: true` avec `DiffOld` et `DiffNew` pour qu'il montre une modification sous forme de diff, `ID` sur une session restaurée pour que l'agent la reconnaisse (`Turn.SessionID` ; ori-agent en donne un aux autres, voir `NewSessionID`), `Busy: true` sur une session pour l'indicateur d'activité, `Open: false` sur un projet pour le replier.

## Fonctionnalités

`Disable` désactive des parties intégrées ; par défaut, tout est actif.

```go
Disable: oriagent.Features{
	Voice:           true, // bouton micro et /voice
	Pomodoro:        true, // minuteur de la barre de titre
	Attachments:     true, // trombone et /upload
	Preview:         true, // panneau d'aperçu
	Setup:           true, // page Configuration (engrenage)
	Profile:         true, // page Profil
	ThemeToggle:     true, // bouton de thème et /theme
	BuiltinCommands: true, // toutes les commandes intégrées
},
```

## Dictée vocale

La dictée fournie est simulée. Pour un vrai moteur de reconnaissance vocale, implémentez :

```go
type Dictation interface {
	Start() error           // commencer la capture
	Stop() (string, error)  // arrêter et renvoyer le texte (peut bloquer)
}
```

En option, `Level() float32` (interface `Leveler`) anime l'onde avec le vrai niveau sonore, et `SetLocale(string)` (interface `LocaleSetter`) suit la langue de l'interface.

## Pomodoro

`Pomodoro: oriagent.PomodoroConfig{Focus: 50 * time.Minute, Short: 10 * time.Minute, Long: 30 * time.Minute}` change les durées (25, 5 et 15 minutes par défaut).

## État au démarrage

Utile pour les démonstrations et les tests :

| Champ | Effet |
|---|---|
| `StartView` | Vue affichée : `"chat"`, `"setup"`, `"profile"` ou l'`ID` d'une page |
| `SetupStep` | Étape de la Configuration (1 à 4) avec `StartView: "setup"` |
| `Attach` | Fichiers joints à la zone de saisie |
| `Preview` | Fichier ouvert dans l'aperçu |
| `Locale`, `LightMode` | Langue et thème |
| `Settings` | Tous les réglages de la Configuration (voir [Mémoriser les réglages](#mémoriser-les-réglages)) |
