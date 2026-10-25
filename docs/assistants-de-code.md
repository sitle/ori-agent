# Utiliser ori-agent avec un assistant de code

Les assistants de code (Claude Code, Codex, Cursor, Copilot…) écrivent de meilleures applications ori-agent quand on leur donne une documentation pensée pour eux : les règles à respecter, l'API exacte et les erreurs à éviter, sans les explications destinées aux humains. Le dépôt en fournit trois, en anglais, la langue que ces outils attendent et celle du code.

| Fichier | Pour qui | Contenu |
|---|---|---|
| [`skills/ori-agent/SKILL.md`](../skills/ori-agent/SKILL.md) | un assistant qui **utilise** la bibliothèque dans votre projet | guide complet : règles d'or, API `Agent`/`Turn`, toutes les options de `Config`, commandes, pages, traductions, erreurs fréquentes, vérifications |
| [`llms.txt`](../llms.txt) | un assistant qui découvre le projet en ligne | index court au format [llms.txt](https://llmstxt.org), avec des liens directs vers les fichiers bruts |
| [`AGENTS.md`](../AGENTS.md) (et `CLAUDE.md`) | un assistant qui **modifie** le framework lui-même | commandes, organisation du code, règles d'architecture et d'écriture |

## Dans votre projet

### Claude Code

Copiez la skill dans votre projet : Claude Code la chargera dès que la tâche concerne ori-agent.

```bash
mkdir -p .claude/skills/ori-agent
curl -L -o .claude/skills/ori-agent/SKILL.md https://raw.githubusercontent.com/sitle/ori-agent/main/skills/ori-agent/SKILL.md
```

Pour l'avoir dans tous vos projets, copiez-la plutôt dans `~/.claude/skills/ori-agent/`.

### Autres assistants

La plupart lisent un fichier d'instructions à la racine du projet (`AGENTS.md`, `.cursorrules`, `.github/copilot-instructions.md`…). Ajoutez-y une ligne qui pointe vers la skill :

```markdown
This project uses the ori-agent framework. Before writing UI or agent code, read
https://raw.githubusercontent.com/sitle/ori-agent/main/skills/ori-agent/SKILL.md
```

Vous pouvez aussi copier le contenu de `SKILL.md` (sans l'en-tête entre `---`) directement dans ce fichier.

### Par une simple demande

Avec un assistant qui peut lire le web, il suffit souvent de lui donner l'index : « Lis https://raw.githubusercontent.com/sitle/ori-agent/main/llms.txt puis crée une application ori-agent qui… ».

## Exemples de demandes

- « Crée une application ori-agent avec un agent qui répond en streaming et une commande /export qui enregistre la conversation dans un fichier Markdown. »
- « Branche l'agent sur l'API de mon fournisseur de LLM, avec la réflexion et les appels d'outils. »
- « Ajoute une page Statistiques traduite en français et en anglais. »
- « Désactive la dictée et le Pomodoro, et passe l'interface aux couleurs de notre charte. »

## Garder ces fichiers à jour

Toute modification de l'API publique doit mettre à jour `SKILL.md` en même temps que la documentation humaine : c'est une règle de [`AGENTS.md`](../AGENTS.md) et de [CONTRIBUTING.md](../CONTRIBUTING.md). Les exemples compilés de `example_test.go` garantissent que les extraits principaux restent valides.
