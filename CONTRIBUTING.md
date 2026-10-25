# Contribuer à ori-agent

Merci de votre intérêt ! Les contributions sont les bienvenues : signalements de bugs, idées, traductions, code.

## Signaler un problème

Ouvrez une [issue](https://github.com/sitle/ori-agent/issues) en précisant :

- le système (Windows, macOS, Linux et sa version) et la version de Go ;
- ce que vous avez fait, ce que vous attendiez, ce qui s'est passé ;
- si possible, une capture d'écran.

## Proposer une modification

1. Forkez le dépôt et créez une branche à partir de `main`.
2. Faites vos modifications, avec des tests quand c'est possible.
3. Vérifiez que tout passe :

   ```bash
   gofmt -l .
   go vet ./...
   go test ./...
   ```

4. Lancez l'exemple complet pour vérifier le rendu : `go run ./examples/anna`.
5. Ouvrez une pull request qui explique le changement et sa raison. Ajoutez une ligne dans `CHANGELOG.md`, section « Non publié ».

## Conventions

- Code formaté par `gofmt`, commentaires en anglais, comme le reste du code.
- Aucun texte d'interface en dur : passez par les catalogues (`locales.go`), en français **et** en anglais. Le test `TestCatalogsHaveSameKeys` vérifie que les deux langues ont les mêmes clés.
- Les couleurs viennent de `theme.Palette`, jamais de valeurs en dur dans les widgets.
- Le paquet `ui` ne doit pas dépendre de la coquille `oriagent`.
- Texte français avec tous ses accents, et de vrais tirets (`-`), jamais de tirets longs.
- Toute modification de l'API publique met à jour, en même temps, `docs/`, la skill [`skills/ori-agent/SKILL.md`](skills/ori-agent/SKILL.md) (lue par les assistants de code) et, si besoin, les exemples compilés de `example_test.go`.

Les assistants de code qui contribuent suivent [AGENTS.md](AGENTS.md).

## Ajouter une langue

Copiez `catalogFR` de `locales.go` dans un nouveau catalogue, traduisez chaque texte (en gardant les verbes `%s`, `%d` dans le même ordre), puis ajoutez-le à `Catalogs`. Faites de même pour `sim/locales.go`.

## Licence

En contribuant, vous acceptez que votre contribution soit publiée sous la licence du projet, l'[EUPL-1.2](LICENSE).
