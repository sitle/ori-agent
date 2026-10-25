# Architecture

Ce document explique comment ori-agent fonctionne à l'intérieur. Il s'adresse à celles et ceux qui veulent contribuer ou comprendre un comportement.

## Les paquets

```
github.com/sitle/ori-agent      la coquille d'application (paquet oriagent)
├── ui        kit de composants      → theme
├── theme     palettes de couleurs
├── i18n      traductions            (aucune dépendance)
├── preview   aperçu de documents    → ui, i18n
├── sim       agent simulé           → oriagent, i18n
└── examples  minimal, anna
```

Règles de dépendance : `ui`, `theme`, `i18n` et `preview` ne dépendent jamais de la coquille. `sim` n'utilise que son API publique, comme le ferait un projet tiers.

## La boucle de rendu

ori-agent repose sur [Gio](https://gioui.org), une bibliothèque d'interface en mode immédiat : à chaque image, tout l'écran est redécrit à partir de l'état. Il n'y a pas d'arbre de widgets à synchroniser.

`App.loop` reçoit les événements de la fenêtre. Pour chaque image :

1. `frame` prend le verrou de l'application ;
2. il traite les événements de l'image précédente : clics, touches, glissement du séparateur, palette de commandes ;
3. il dessine la barre de titre, puis le corps : barre latérale, vue courante (conversation, Configuration, Profil ou page), aperçu ;
4. il dessine le toast, puis rend la main à Gio.

Les animations (pastilles qui pulsent, réflexion en cours, onde de la dictée) demandent simplement une nouvelle image.

## Le modèle de concurrence

Tout l'état de l'application (projets, sessions, messages, réglages) est protégé par un seul verrou, `App.mu`.

- Le rendu tient le verrou pendant toute l'image.
- Les commandes s'exécutent pendant le rendu : elles modifient l'état directement, sans verrou à prendre (d'où la règle « `Ctl` n'est valable que pendant `Run` »).
- Les travaux de fond (agent, envoi simulé de fichiers, dictée, rendu PDF) passent par `App.update(f)` : prendre le verrou, appliquer `f`, relâcher, demander une image.
- Chaque méthode de `Turn` est un `update` : un agent peut donc écrire depuis n'importe quelle goroutine sans rien savoir du verrou.
- L'aperçu PDF a son propre verrou, pour rester utilisable hors de la coquille.

## Le cycle d'un tour

1. L'utilisateur envoie un message : `onSubmit` ajoute la bulle, puis `startTurn` crée un contexte annulable, marque la session occupée et lance l'agent dans une goroutine.
2. L'agent écrit par le `Turn` ; chaque message qu'il crée est mémorisé.
3. À la fin, ou si le bouton Arrêter annule le contexte, la session est libérée. Tout ce qui est resté « en cours » est refermé (réflexions, appels d'outils, sous-agents, tâches), puis « Interrompu. » ou l'erreur s'affiche.

## Les données

- Un `Project` contient des `Session`, et une session des `Children` (ses sous-agents) : c'est l'arborescence de la barre latérale.
- Une session contient des `Message`, de plusieurs sortes (`Kind`) : bulle utilisateur, texte, réflexion, appel d'outil, tâches, commande, avis.
- L'état visuel (boutons, dépliement) est stocké à côté des données, dans des champs non exportés.

## Les textes

Aucun texte d'interface n'est écrit en dur : tout passe par `App.t(clé)`, qui interroge le `i18n.Bundle` de l'application (catalogues intégrés de [`locales.go`](../locales.go), puis `Config.Translations`). Comme l'écran est redécrit à chaque image, changer de langue ne demande aucune mise à jour : l'image suivante est dans la nouvelle langue.

## La fenêtre

La fenêtre est créée sans décoration système (`app.Decorated(false)`). ori-agent dessine sa barre de titre ; la faire glisser déplace la fenêtre (`system.ActionMove`), et les boutons réduire, agrandir et fermer appellent les actions de Gio.

## Tests

`go test ./...` vérifie notamment que les catalogues français et anglais ont les mêmes clés et les mêmes paramètres, que les commandes intégrées sont traduites, le parseur Markdown, les tailles de fichier et le cycle du Pomodoro. Les exemples de [`example_test.go`](../example_test.go) sont compilés à chaque test : ils ne peuvent pas devenir faux sans que la compilation échoue.
