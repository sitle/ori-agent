# Composants et thèmes

ori-agent est découpé en paquets indépendants. Trois d'entre eux servent aussi en dehors de la coquille d'application, dans n'importe quel programme [Gio](https://gioui.org).

## Le kit `ui`

`github.com/sitle/ori-agent/ui` regroupe les composants dessinés par ori-agent. Il ne dépend pas de la coquille.

```go
th := ui.NewTheme(theme.OriLight, theme.OriDark) // démarre en sombre
th.SetDark(false)
```

Le `Theme` porte la palette active (`th.Bg`, `th.Text`, `th.Accent`…) et le moteur de texte. Changer de thème remplace seulement la palette : tout ce qui est dessiné ensuite suit.

| Famille | Fonctions et méthodes |
|---|---|
| Texte | `th.Label`, `th.Mono`, `th.Body`, `ui.OneLine` (une ligne coupée par « … ») |
| Boutons | `th.Button` (`ui.Primary`, `ui.Secondary`, `ui.Danger`), `th.IconButton`, `th.RoundButton`, `th.NavRow` |
| Formulaires | `th.Field`, `th.ToggleRow`, `th.Switch`, `th.Radio`, `th.ChoiceCard` |
| Mise en page | `th.Page`, `th.Section`, `th.Panel`, `ui.Card`, `ui.Bordered`, `ui.Centered`, `ui.Flow`, `ui.HSpace`, `ui.VSpace`, `ui.Expand`, `th.HLine` |
| Surcouches | `ui.Popover` (au-dessus de tout, fermé par un clic à côté), `th.Menu`, `ui.OverlayAbove`, `ui.Toast` |
| Indicateurs | `ui.Dot`, `ui.Ring`, `ui.Pulse` (animation « en cours »), `ui.ProgressRing`, `ui.Waveform`, `th.Chip`, `th.Avatar`, `th.CodeLines` |
| Icônes | `ui.Chevron`, `ui.Close`, `ui.Gear`, `ui.Sidebar`, `ui.Mic`, `ui.Stop`, `ui.Paperclip`, `ui.WindowIcon` |
| Logo | `ui.NewLogo(svg)`, rastérisé à la taille exacte, net sur tous les écrans |

Exemple, un panneau de réglages :

```go
func reglages(gtx layout.Context, th *ui.Theme, notif *widget.Bool, ok *widget.Clickable) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return th.Section(gtx, "Notifications", "Quand être prévenu.", func(gtx layout.Context) layout.Dimensions {
				return th.ToggleRow(gtx, notif, "Fin de tâche", "Prévenir quand une tâche longue se termine.")
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return th.Button(gtx, ok, "Enregistrer", ui.Primary)
		}),
	)
}
```

## Changer de design system

Une palette est une structure `theme.Palette` d'une vingtaine de couleurs nommées par leur rôle : fonds (`Bg`, `Sidebar`, `Surface`, `Hover`, `Selected`), bordures, textes, accent, retours (succès, ajouts et suppressions de diff), aplats (`Danger`, `Warning`, `SuccessStrong`, `OnStrong`), bulle utilisateur et code. Fournissez-en deux :

```go
var maMarqueClaire = theme.Palette{
	Bg: theme.RGB(0xffffff), Text: theme.RGB(0x1d1d1f), Accent: theme.RGB(0x00875a),
	// … toutes les couleurs
}

oriagent.Run(oriagent.Config{Light: maMarqueClaire, Dark: maMarqueSombre})
```

Partez de `theme.OriLight` et `theme.OriDark` dans [`theme/theme.go`](../theme/theme.go) : chaque couleur y est commentée avec le jeton Ori dont elle vient. Aucun widget n'utilise de couleur en dur.

Les polices se changent avec `Config.Fonts` (ou le dernier paramètre de `ui.NewTheme`) ; `th.MonoFace` choisit la police à chasse fixe.

## L'aperçu de documents `preview`

`github.com/sitle/ori-agent/preview` affiche un document dans un panneau, avec son en-tête :

```go
doc := preview.Open("rapport.pdf", "")  // nil si illisible
doc.Load(window.Invalidate)             // rendu des pages PDF en arrière-plan
// dans la boucle de rendu :
doc.Layout(gtx, th)
if doc.Close.Clicked(gtx) { /* fermer */ }
```

- **Markdown** : titres, paragraphes, listes, blocs de code, tableaux, séparateurs.
- **Code** : numéros de ligne, police à chasse fixe.
- **PDF** : rendu par PDFium compilé en WebAssembly, en Go pur, 50 pages au plus.
- **Images** : PNG, JPEG, GIF.

`doc.T` reçoit un traducteur pour les textes du panneau ; sans traducteur, ils sont en anglais.
