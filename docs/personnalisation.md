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
| `Models()`, `Model()`, `SetModel(i)` | Modèle actif |
| `Dark()`, `SetDark(b)` | Thème |
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

Les modifications faites dans la page Profil restent en mémoire.

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

Mettez `Expanded: true` sur un message d'outil pour l'afficher déplié, `Busy: true` sur une session pour l'indicateur d'activité, `Open: false` sur un projet pour le replier.

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
