# Journal des modifications

Le format suit [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/) et le projet respecte le [versionnage sémantique](https://semver.org/lang/fr/). Tant que la version reste en 0.x, l'API peut changer d'une version mineure à l'autre.

## [Non publié]

## [0.2.2] - 2026-10-26

### Ajouté

- `Config.ProjectFiles` et `ProjectFile` : section « Fichiers » de la page d'un projet, remplie par l'application (par exemple les fichiers joints dans ses sessions) avec, pour chaque fichier, un détail (taille, date, version, empreinte…) et un avertissement en rouge ; un clic ouvre l'aperçu. La liste est demandée dans une goroutine à l'ouverture de la page.
- L'exemple Anna liste les fichiers du dossier de chaque projet, avec leur taille dans la langue de l'interface, leur date et un avertissement au-delà de 20 Mo.
- Capture de la page d'un projet dans la documentation ; captures du README mises à jour.

### Corrigé

- L'aperçu s'affiche aussi à côté de la page d'un projet, et plus seulement à côté de la conversation.
- Un fichier introuvable dans la section « Fichiers » affiche une notification au lieu de ne rien faire.

## [0.2.1] - 2026-10-26

### Corrigé

- Les tableaux Markdown des réponses et de l'aperçu sont mis en forme au lieu d'être affichés en texte brut : en-tête sur fond teinté, filet entre les lignes, alignement des colonnes (`:---:`, `---:`), texte coupé dans sa cellule quand le tableau est plus large que la place disponible (chaque colonne garde au moins la largeur de son mot le plus long), `\|` dans une cellule.

### Ajouté

- `ui.SplitRow` et `ui.ParseTable` : lecture d'un tableau Markdown (lignes, alignements, présence d'un en-tête).

## [0.2.0] - 2026-10-25

### Ajouté

- `Settings`, `Config.Settings` et `Config.OnSettings` : l'application restaure au démarrage les réglages de la page Configuration (dossier de travail, autorisations, réflexion étendue, notifications, modèle, thème, langue) et reçoit les nouvelles valeurs à chaque changement pour les enregistrer. `DefaultSettings()` donne les valeurs par défaut.
- `Ctl.Settings()` : les réglages courants, pour les commandes.
- `Turn.AllowCommands` et `Turn.AskEdits` : les autorisations de l'espace de travail, pour les agents.
- `Config.OnProfile` et `Config.OnLogout` : l'application enregistre le profil et gère la déconnexion ; ils s'exécutent dans une goroutine, les boutons de la page Profil attendent la fin, une erreur s'affiche dans une notification.
- `Config.ProfileLock` : champs du profil en lecture seule.
- `Turn.SetUser`, `Turn.SetModels`, `Ctl.User`, `Ctl.SetUser`, `Ctl.SetModels` : remplacer le profil et la liste des modèles après le démarrage, par exemple après une connexion.
- Rendu Markdown des réponses de l'assistant : titres, paragraphes, listes à puces (imbriquées) et numérotées, citations, blocs de code, tableaux, filets ; en ligne, gras, italique, code et liens. Un marqueur encore ouvert pendant le streaming reste affiché tel quel.
- `ui.ParseMarkdown`, `ui.ParseInline`, `ui.PlainInline`, `Theme.Markdown`, `Theme.MarkdownBlock` et `Theme.RichText` (texte à styles mélangés, coupé entre les mots), utilisables hors de l'application.
- `Project.WorkDir` et `Project.Description` : un projet peut avoir son propre dossier, transmis à l'agent dans `Turn.WorkDir` pour ses sessions.
- `Config.OnNewProject` : le bouton principal de la barre latérale ouvre un formulaire (nom, description) et l'application crée le projet, par exemple un dossier d'espace de travail ; l'appel se fait dans une goroutine, une erreur reste affichée dans le formulaire.
- Bouton « + » à droite de chaque projet de la barre latérale : démarre une session dans ce projet (ou reprend sa session vide). `Ctl.NewSessionIn` fait de même depuis une commande.
- Page d'un projet, ouverte par la roue dentée de sa ligne quand `Config.OnUpdateProject` est défini : nom, description, autorisations, dossier (copiable) ; `Config.OnDeleteProject` ajoute un bouton de suppression à confirmer, absent pour un projet `Permanent`.
- `Permissions` et `Project.Permissions` : autorisations propres à un projet, transmises à l'agent dans `Turn.AllowCommands` et `Turn.AskEdits` (sinon celles de la Configuration).
- Le dossier affiché (accueil, zone de saisie), copié, ouvert ou proposé pour joindre un fichier est celui du projet de la session ; `Ctl.WorkDir` et `Ctl.Open` le suivent aussi.

### Modifié

- L'aperçu des fichiers Markdown utilise le même rendu : le gras, l'italique, le code et les liens y sont désormais affichés au lieu d'être retirés.
- L'exemple `anna` utilise les nouveautés : il enregistre les réglages, le profil et les projets créés dans le dossier de configuration de l'utilisateur (option `-data` pour en choisir un autre), verrouille l'e-mail du profil, propose `/login` (profil et modèles remplacés après une connexion simulée) et donne aux projets de démo un dossier, une description, des autorisations propres ou le statut permanent. Les options `-lang` et `-light` restent prioritaires sur les réglages enregistrés.
- Documentation mise à jour : projets, réglages, profil et Markdown dans le README, les guides et la skill des assistants de code.

### Changements incompatibles

- `preview` ne fournit plus son propre parseur Markdown : il utilise celui de `ui` (`ui.ParseMarkdown`).
- Quand `Config.Settings` est fourni, il l'emporte sur `WorkDir`, `LightMode` et `Locale`.

## [0.1.0] - 2026-10-24

Première version publique.

### Ajouté

- Coquille d'application `oriagent` : barre de titre dessinée par l'app, arborescence projets → sessions → sous-agents, fil de conversation, zone de saisie, aperçu redimensionnable, pages Configuration et Profil.
- Interface `Agent` et objet `Turn` : texte et réflexion en streaming, appels d'outils, listes de tâches, sous-agents, notifications, annulation par le bouton Stop.
- Commandes `/` avec palette, extensibles par `Config.Commands` ; commandes intégrées `/help`, `/new`, `/clear`, `/compact`, `/model`, `/open`, `/usage`, `/status`, `/lang`, `/sidebar`, `/upload`, `/voice`, `/pomodoro`, `/theme`, `/config`, `/profile`.
- Pages personnalisées (`Config.Pages`), modèles, profil, données de démonstration.
- Dictée vocale (interface `Dictation`, version simulée fournie), minuteur Pomodoro, pièces jointes avec sélecteur de fichiers natif.
- Paquet `ui` : kit de composants Gio réutilisable seul.
- Paquet `theme` : palettes du design system Ori, claire et sombre.
- Paquet `preview` : aperçu Markdown, code, PDF (PDFium en WebAssembly) et images.
- Paquet `i18n` : langues de l'interface, catalogues français et anglais complets, changement en direct.
- Paquet `sim` : agent simulé bilingue.
- Exemples `examples/minimal` et `examples/anna`, exemples compilés dans `example_test.go`.
- Documentation en français dans `docs/` : prise en main, écrire un agent, personnalisation, langues, composants, architecture.
- Documentation pour les assistants de code : skill `skills/ori-agent/SKILL.md`, `llms.txt`, `AGENTS.md`.

[Non publié]: https://github.com/sitle/ori-agent/compare/v0.2.2...HEAD
[0.2.2]: https://github.com/sitle/ori-agent/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/sitle/ori-agent/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/sitle/ori-agent/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/sitle/ori-agent/releases/tag/v0.1.0
