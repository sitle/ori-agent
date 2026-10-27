# Journal des modifications

Le format suit [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/) et le projet respecte le [versionnage sémantique](https://semver.org/lang/fr/). Tant que la version reste en 0.x, l'API peut changer d'une version mineure à l'autre.

## [Non publié]

## [0.8.0] - 2026-10-26

### Ajouté

- Panneaux de l'application : `Config.Panels` (`Panel{ID, Title, Icon, Layout}`) ajoute des boutons dans la barre de titre ; chacun ouvre, à droite du fil, un panneau qui dessine ce que l'application veut sur la session affichée (`Layout(gtx, th, tr, session)`), dans une colonne défilante avec un titre et un bouton de fermeture. Le panneau prend la place de l'aperçu, et l'aperçu la sienne ; il est redessiné deux fois par seconde tant qu'un tour tourne dans la session. `Ctl.TogglePanel(id)` l'ouvre ou le ferme depuis une commande ; un identifiant inconnu, ou un panneau sans `Layout`, ne fait rien.
- Exemple Anna : panneau « Supervision » (état, mode, outils, contexte, tâches, sous-agents), écrit avec l'API publique.
- Capture du panneau dans la documentation ; captures mises à jour avec le nouveau bouton de la barre de titre.

### Corrigé

- Les boutons d'une demande d'approbation passent à la ligne quand la carte est étroite (aperçu ou panneau ouvert), au lieu de couper leur texte.
- L'aperçu des PDF fonctionne dans un exécutable sans console (`-ldflags=-H=windowsgui`) : PDFium n'écrit plus sur une sortie standard absente.

### Corrigé

- Aperçu des PDF : dans une application construite sans console (`-H=windowsgui` sous Windows), le rendu échouait avec « could not instantiate webassembly module: GetFileType /dev/stdout ». PDFium n'écrit plus sur la sortie standard.

## [0.7.0] - 2026-10-26

### Ajouté

- Message pendant un tour : la zone de saisie reste utilisable pendant que l'agent travaille ; le bouton d'envoi devient « Ajouter » et le message s'affiche aussitôt, marqué « En attente ». L'agent le reçoit à sa prochaine étape avec `Turn.TakeMessages()` (`QueuedMessage{Text, Attachments}`) ; s'il ne le prend pas, un nouveau tour part à la fin du sien ; « Arrêter » remet le texte en attente dans la zone de saisie. `Headless.Send` fait de même sans fenêtre.
- Retour à un message : `Config.OnRewind(ctx, session, n)` ajoute « ↶ Revenir ici » au survol des messages de l'utilisateur, avec une confirmation ; l'application rétablit son état d'avant le n-ième message (0 pour le premier), puis le fil est coupé à ce message et son texte revient dans la zone de saisie. Une erreur s'affiche et laisse le fil tel quel.
- Mention de fichiers : `Config.Mentions(session, requête)` ouvre, à la frappe de « @mot », une liste de suggestions (`Mention{Label, Detail, Insert}`, 8 au plus) ; flèches pour choisir, Tab ou Entrée pour insérer « @chemin », Échap pour fermer.
- Exemple Anna : « @ » propose les fichiers du dossier de la session, le retour à un message est actif, et l'agent simulé prend les messages envoyés pendant son tour avant de répondre.
- Captures de la liste « @ » et d'un message en attente dans la documentation ; captures mises à jour.
- Un message en attente qui part comme nouveau tour en est le texte et ne figure pas en plus dans l'historique transmis à l'agent.
- « Arrêter » remet aussi les pièces jointes en attente dans la zone de saisie, après ce qu'elle contient déjà ; si une autre session est affichée, les messages restent dans le fil.
- Pendant un retour à un message, aucun tour ne peut partir dans la session ; les sous-agents lancés après le message sont retirés, et ses pièces jointes reviennent avec son texte.
- Échap ferme la liste « @ » sans modifier le texte, et le bouton Envoyer envoie le message au lieu d'insérer une suggestion.

## [0.6.0] - 2026-10-26

### Ajouté

- Modes de travail : `Config.Modes` (`Mode{ID, Label, Desc}`) transforme le badge de la zone de saisie en menu ; le mode est gardé par projet (`Project.Mode`), transmis à l'agent (`Turn.Mode`, `TurnOptions.Mode`) et enregistré par `Config.OnMode` (une erreur remet le mode précédent). `Turn.SetMode`, `Ctl.Mode` et `Ctl.SetMode` le lisent et le changent sans appeler `OnMode`.
- Jauge de contexte dans la zone de saisie : `Turn.SetContext(utilisés, fenêtre)`, pourcentage et anneau orange à 70 %, rouge à 90 % ; `Session.ContextUsed` et `ContextWindow` pour la restaurer.
- Approbation : `ApprovalRequest.AllowLabel` et `DenyLabel` changent les libellés des boutons, `Markdown` affiche le détail comme un texte mis en forme (un plan à valider).
- Agent simulé : `sim.Modes` (Édition, Plan, Auto) ; en mode Plan, il fait valider un plan avant d'agir puis passe en Édition ; en Auto, il modifie sans demander ; il remplit la jauge de contexte. L'exemple Anna utilise ces modes et garde celui de chaque projet créé.
- Capture d'un plan à valider dans la documentation ; captures mises à jour avec le nouveau badge.

## [0.5.1] - 2026-10-26

### Corrigé

- Barre latérale : les boutons des pages de l'application (`Config.Pages`) ne touchent plus le filet au-dessus du profil ; deux pages sont séparées d'un petit espace.
- Captures de la documentation mises à jour.

## [0.5.0] - 2026-10-26

### Ajouté

- `Config.Convert` et `preview.Converter` : l'application fournit l'aperçu des fichiers que le panneau ne lit pas lui-même (Word, Excel, PowerPoint…), sous forme de PDF ou de Markdown. La conversion se fait en arrière-plan, avec « Conversion du document… » en attendant ; l'en-tête du panneau montre le libellé donné (« Word · 3 p. ») et, pendant la conversion, l'extension du fichier (« DOCX »). `preview.OpenWith` fait de même hors de l'application.
- L'exemple Anna affiche les documents Word (`.docx`) convertis en Markdown, sans dépendance ; document de démonstration `examples/anna/docs/note-de-cadrage.docx`.
- Capture de l'aperçu d'un document Word dans la documentation.

### Corrigé

- Un texte de plus de 256 Ko dont la coupure tombait au milieu d'un caractère accentué était affiché comme un fichier binaire.

## [0.4.0] - 2026-10-26

### Ajouté

- `Headless`, `NewHeadless` et `TurnOptions` : faire tourner l'agent sans fenêtre, par le même `Turn` que l'application, pour des tests, des bancs d'évaluation ou des scripts. `Headless.Run` lance un tour dans une session (nouvelle ou existante) et rend la session, dont le fil se lit ensuite ; `Headless.Approve` répond aux demandes d'approbation (refus si absent), dans la goroutine de l'agent. Un second tour dans une session occupée renvoie `ErrSessionBusy`.
- Test de l'agent simulé sans fenêtre (`sim/sim_test.go`) et exemple compilé `ExampleNewHeadless`.
- Documentation : « Faire tourner un agent sans fenêtre » dans `docs/agents.md`, section correspondante dans la skill.

## [0.3.0] - 2026-10-26

Cette version reprend le contenu de la 0.2.3, publiée par erreur avec un numéro de correctif et retirée (`retract` dans `go.mod`).

### Ajouté

- `Session.ID` et `Turn.SessionID` : identifiant stable de la session, donné par ori-agent aux nouvelles sessions (et aux sous-agents), fourni par l'application pour les sessions qu'elle restaure ; `NewSessionID()`. L'agent retrouve ainsi ses propres enregistrements sans dépendre du texte affiché.
- `Turn.Approve`, `ApprovalRequest` et `Decision` : l'agent demande une approbation (commande, modification de fichier…) et attend la réponse. Une carte s'affiche dans le fil avec le détail, le diff éventuel et les boutons Autoriser, Toujours pour cette session (si proposé) et Refuser ; Stop retire la demande. La demande d'un sous-agent s'affiche dans la session principale, en le nommant ; une notification la signale si une autre session est affichée.
- Vue de diff : `ToolCall.Diff(ancien, nouveau)` montre dans la carte d'un appel d'outil les lignes retirées et ajoutées, avec numéros de ligne, trois lignes de contexte et le décompte « +3 -1 » dans l'en-tête ; `ui.DiffLines`, `ui.Hunks`, `ui.DiffStat` et `Theme.Diff` sont utilisables hors de l'application. `Message.HasDiff`, `DiffOld` et `DiffNew` font de même dans un transcript écrit d'avance.
- L'agent simulé (`sim`, exemple Anna) propose une modification avec son diff, sans l'écrire sur le disque, et demande d'abord une approbation quand les autorisations l'exigent ; « Toujours pour cette session » est retenu par `Turn.SessionID`. Les modifications du transcript de démonstration s'affichent en diff.
- Une demande d'approbation laissée sans réponse à la fin d'un tour est marquée annulée.
- Capture d'une demande d'approbation dans la documentation ; captures du README mises à jour.

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

[Non publié]: https://github.com/sitle/ori-agent/compare/v0.8.0...HEAD
[0.8.0]: https://github.com/sitle/ori-agent/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/sitle/ori-agent/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/sitle/ori-agent/compare/v0.5.1...v0.6.0
[0.5.1]: https://github.com/sitle/ori-agent/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/sitle/ori-agent/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/sitle/ori-agent/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/sitle/ori-agent/compare/v0.2.2...v0.3.0
[0.2.2]: https://github.com/sitle/ori-agent/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/sitle/ori-agent/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/sitle/ori-agent/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/sitle/ori-agent/releases/tag/v0.1.0
