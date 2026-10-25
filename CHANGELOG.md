# Journal des modifications

Le format suit [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/) et le projet respecte le [versionnage sémantique](https://semver.org/lang/fr/). Tant que la version reste en 0.x, l'API peut changer d'une version mineure à l'autre.

## [Non publié]

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

[Non publié]: https://github.com/sitle/ori-agent/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/sitle/ori-agent/releases/tag/v0.1.0
