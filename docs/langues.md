# Langues de l'interface

Tous les textes de l'interface viennent de catalogues de traduction. Le français et l'anglais sont fournis en entier ; vous pouvez ajouter vos textes et d'autres langues.

## Choisir la langue

- **Au démarrage** : `Config.Locale` (`"fr"`, `"en"`, `"en-US"`…). Vide, c'est la langue du système si un catalogue existe, sinon la langue de repli.
- **Dans l'application** : à l'étape Préférences de la Configuration, ou avec la commande `/lang en` (sans argument, elle liste les langues).
- **Depuis le code** : `c.SetLocale("en")` dans une commande.

Le changement est immédiat, sans redémarrage. Les messages déjà échangés restent dans leur langue d'origine : ce sont des contenus, pas des textes d'interface. `Config.OnLocale` est appelé après chaque changement, par exemple pour mémoriser le choix ; `Config.OnSettings` aussi, avec tous les réglages (voir [Mémoriser les réglages](personnalisation.md#mémoriser-les-réglages)).

## Traduire vos textes

Les textes que vous fournissez à `Config` (`Model.Desc`, `Page.Title`, `Command.Desc` et `Args`, `Placeholder`, `ModeLabel`, `EmptyTitle`, `EmptySubtitle`) sont cherchés dans les catalogues. S'il n'existe pas de traduction, ils s'affichent tels quels : vous pouvez donc commencer sans traduire, puis remplacer les textes par des clés.

```go
oriagent.Run(oriagent.Config{
	Translations: map[string]map[string]string{
		"fr": {"cmd.export.desc": "Exporter la conversation"},
		"en": {"cmd.export.desc": "Export the conversation"},
	},
	Commands: []*oriagent.Command{{Name: "export", Desc: "cmd.export.desc", Run: exporter}},
})
```

Où traduire dans votre code :

| Endroit | Comment |
|---|---|
| Une page | `tr.T("clé")` (paramètre de `Layout`) |
| Une commande | `c.T("clé")` |
| Un agent | `t.T("clé")`, dans la langue du tour ; `t.Locale` pour choisir la langue de la réponse |
| Ailleurs | un `*i18n.Bundle` à vous, avec `i18n.New` |

Pour combiner plusieurs jeux de traductions, par exemple les vôtres et ceux de l'agent simulé : `i18n.Merge(sim.Translations, mesTextes)`.

## Paramètres dans les textes

Un texte peut contenir des verbes `fmt` (`%s`, `%d`, `%.1f`…), remplis par les arguments de `T` :

```go
"toast.saved": "Fichier %s enregistré (%d lignes)",
```

```go
c.T("toast.saved", nom, n)
```

Une traduction doit garder les verbes dans le même ordre, ou les numéroter : `"%[2]d lignes enregistrées dans %[1]s"`.

## Ajouter une langue

1. Copiez le catalogue français (`catalogFR` dans [`locales.go`](../locales.go)) et traduisez chaque texte. Donnez à la clé `lang.name` le nom de la langue dans cette langue (par exemple « Español »).
2. Passez-le dans `Config.Translations` sous son code (`"es"`).
3. La langue apparaît dans la Configuration et dans `/lang`. Les clés oubliées s'affichent dans la langue de repli (`Config.FallbackLocale`, `"fr"` par défaut).

Pour l'agent simulé, faites de même avec [`sim/locales.go`](../sim/locales.go). Pour que la langue soit intégrée au framework, proposez une contribution : le test `TestCatalogsHaveSameKeys` vérifie qu'aucune clé ne manque.

## Formats

Les tailles de fichier suivent la langue (« 1,2 Mo », « 1.2 MB ») : utilisez `oriagent.HumanSize(n, tr)`, où `tr` peut être un `Turn` ou un `Ctl`.

## Le paquet i18n seul

`github.com/sitle/ori-agent/i18n` ne dépend de rien d'autre et peut servir dans n'importe quel programme :

```go
b := i18n.New("fr")                  // langue de repli
b.Add("fr", map[string]string{"hi": "Bonjour %s"})
b.Add("en", map[string]string{"hi": "Hello %s"})
b.SetLocale(i18n.Detect())           // langue du système
fmt.Println(b.T("hi", "Teva"))
```
