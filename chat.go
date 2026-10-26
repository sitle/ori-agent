package oriagent

import (
	"image"
	"path/filepath"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/widget/material"

	"github.com/sitle/ori-agent/preview"
	"github.com/sitle/ori-agent/ui"
)

const chatWidth = 820

func (a *App) layoutChat(gtx C) D {
	s := a.cur
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Flexed(1, func(gtx C) D {
			if len(s.Messages) == 0 {
				return a.emptyState(gtx)
			}
			return material.List(a.th.Material, &a.transcript).Layout(gtx, len(s.Messages), func(gtx C, i int) D {
				return ui.Centered(gtx, chatWidth, func(gtx C) D {
					in := layout.Inset{Bottom: 18, Left: 28, Right: 28}
					if i == 0 {
						in.Top = 28
					}
					return in.Layout(gtx, func(gtx C) D { return a.message(gtx, s.Messages[i]) })
				})
			})
		}),
		layout.Rigid(a.composer),
	)
}

func (a *App) emptyState(gtx C) D {
	th := a.th
	sub := a.cfg.EmptySubtitle
	if sub == "" {
		sub = a.sessionWorkDir(a.cur)
	}
	return layout.Center.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx C) D { return a.logo.Layout(gtx, 72, th.Accent) }),
			ui.VSpace(16),
			layout.Rigid(th.Label(24, a.t(a.cfg.EmptyTitle), th.Text, font.Medium).Layout),
			ui.VSpace(6),
			layout.Rigid(th.Label(14, a.t(sub), th.Muted, font.Normal).Layout),
		)
	})
}

func (a *App) composer(gtx C) D {
	th := a.th
	return ui.Centered(gtx, chatWidth, func(gtx C) D {
		return layout.Inset{Top: 8, Bottom: 20, Left: 28, Right: 28}.Layout(gtx, func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			if _, open := a.paletteFilter(); open {
				defer ui.OverlayAbove(gtx, 8, a.layoutPalette)
			}
			return ui.Bordered(gtx, th.Surface, th.BorderHi, 14, func(gtx C) D {
				return layout.Inset{Top: 12, Bottom: 10, Left: 14, Right: 10}.Layout(gtx, func(gtx C) D {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							if len(a.attachments) == 0 {
								return D{}
							}
							return layout.Inset{Bottom: 10}.Layout(gtx, func(gtx C) D {
								return a.attachmentFlow(gtx, a.attachments, true)
							})
						}),
						layout.Rigid(func(gtx C) D {
							gtx.Constraints.Min = image.Pt(gtx.Constraints.Max.X, 0)
							gtx.Constraints.Max.Y = gtx.Dp(160)
							ed := material.Editor(th.Material, &a.editor, a.t(a.cfg.Placeholder))
							ed.TextSize = 15
							ed.Color, ed.HintColor = th.Text, th.Muted
							d := ed.Layout(gtx)
							if m := gtx.Dp(30); d.Size.Y < m {
								d.Size.Y = m
							}
							return d
						}),
						ui.VSpace(10), // keeps multi-line text off the bottom row
						layout.Rigid(func(gtx C) D {
							return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx C) D {
									if a.cfg.Disable.Attachments {
										return D{}
									}
									return layout.Inset{Right: 8}.Layout(gtx, a.attachButton)
								}),
								layout.Rigid(func(gtx C) D {
									if a.cfg.ModeLabel == "-" {
										return D{}
									}
									return layout.Inset{Right: 10}.Layout(gtx, func(gtx C) D { return th.Chip(gtx, a.t(a.cfg.ModeLabel), th.Accent) })
								}),
								layout.Flexed(1, func(gtx C) D {
									if a.voice.active() {
										return a.voiceMeter(gtx)
									}
									return ui.OneLine(gtx, th.Mono(12, a.sessionWorkDir(a.cur), th.Muted))
								}),
								ui.HSpace(10),
								layout.Rigid(func(gtx C) D {
									if a.cfg.Disable.Voice {
										return D{}
									}
									return layout.Inset{Right: 8}.Layout(gtx, a.micButton)
								}),
								layout.Rigid(a.sendButton),
							)
						}),
					)
				})
			})
		})
	})
}

// sendButton turns into a stop button while the session is busy.
func (a *App) sendButton(gtx C) D {
	th := a.th
	busy := a.cur.Busy
	return a.send.Layout(gtx, func(gtx C) D {
		bg, txt, fg := th.Accent, a.t("composer.send"), th.OnAccent
		switch {
		case busy:
			bg, txt, fg = th.Selected, a.t("composer.stop"), th.Text
			if a.send.Hovered() {
				bg = th.Hover
			}
		case a.send.Hovered():
			bg = th.AccentHover
		}
		return ui.Card(gtx, bg, 8, layout.Inset{Top: 6, Bottom: 6, Left: 14, Right: 14}, func(gtx C) D {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					if !busy {
						return D{}
					}
					return layout.Inset{Right: 8}.Layout(gtx, func(gtx C) D { return ui.Dot(gtx, ui.Pulse(gtx, th.Accent), 8) })
				}),
				layout.Rigid(th.Label(13, txt, fg, font.SemiBold).Layout),
			)
		})
	})
}

// ---- transcript items ----

func (a *App) message(gtx C, m *Message) D {
	th := a.th
	switch m.Kind {
	case KindUser:
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.End}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				if m.Text == "" {
					return D{}
				}
				return layout.E.Layout(gtx, func(gtx C) D {
					gtx.Constraints.Max.X = gtx.Constraints.Max.X * 4 / 5
					return ui.Card(gtx, th.UserBubble, 14, layout.Inset{Top: 10, Bottom: 10, Left: 14, Right: 14}, th.Body(15, m.Text))
				})
			}),
			layout.Rigid(func(gtx C) D {
				if len(m.Attachments) == 0 {
					return D{}
				}
				return layout.Inset{Top: 6}.Layout(gtx, func(gtx C) D {
					return layout.E.Layout(gtx, func(gtx C) D { return a.attachmentFlow(gtx, m.Attachments, false) })
				})
			}),
		)
	case KindAssistant:
		return th.Markdown(15, m.Text)(gtx)
	case KindThinking:
		return a.thinking(gtx, m)
	case KindTool:
		return a.toolCall(gtx, m)
	case KindTasks:
		return a.taskList(gtx, m)
	case KindApproval:
		return a.approvalCard(gtx, m)
	case KindCommand:
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return ui.Bordered(gtx, th.Bg, th.Border, 8, func(gtx C) D {
					return layout.Inset{Top: 4, Bottom: 4, Left: 10, Right: 10}.Layout(gtx, th.Mono(13, m.Text, th.Accent).Layout)
				})
			}),
			layout.Rigid(func(gtx C) D {
				if m.Output == "" {
					return D{}
				}
				return layout.Inset{Top: 8, Left: 3}.Layout(gtx, func(gtx C) D {
					return ui.Ruled(gtx, th.BorderHi, th.CodeLines(m.Output))
				})
			}),
		)
	case KindNotice:
		c := th.Muted
		if m.Error {
			c = th.DiffDel
		}
		return ui.Ruled(gtx, c, th.Label(13, a.t(m.Text), c, font.Normal).Layout)
	}
	return D{}
}

// isFileTool reports whether a tool call's argument is a file path.
func isFileTool(m *Message) bool {
	return m.FilePath != "" || m.ToolName == "Read" || m.ToolName == "Edit" || m.ToolName == "Write"
}

// openMessage follows a tool call link: subagent session or file preview.
func (a *App) openMessage(m *Message) {
	if m.Sub != nil {
		a.cur, a.focus = m.Sub, true
		return
	}
	fallback := m.Output
	if m.ToolName == "Read" {
		fallback = preview.StripLineNumbers(fallback)
	}
	path := m.FilePath
	if path == "" {
		path = m.ToolArg
		if !filepath.IsAbs(path) {
			path = filepath.Join(a.sessionWorkDir(a.cur), path)
		}
	}
	a.showPreview(a.openDoc(path, fallback))
}

func (a *App) toolCall(gtx C, m *Message) D {
	th := a.th
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			// Siblings rather than nested clickables, so a click on the file
			// name opens the preview without also toggling the output.
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return m.toggle.Layout(gtx, func(gtx C) D {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								if m.Running {
									return ui.Dot(gtx, ui.Pulse(gtx, th.Accent), 8)
								}
								return ui.Dot(gtx, th.Success, 8)
							}),
							ui.HSpace(10),
							layout.Rigid(th.Label(14, m.ToolName, th.Text, font.SemiBold).Layout),
							ui.HSpace(8),
						)
					})
				}),
				layout.Rigid(func(gtx C) D {
					l := th.Mono(13, m.ToolArg, th.Muted)
					if !(isFileTool(m) && !m.Running) && m.Sub == nil {
						return ui.OneLine(gtx, l)
					}
					return m.open.Layout(gtx, func(gtx C) D {
						l.Color = th.Accent
						if m.open.Hovered() {
							l.Color = th.AccentHover
						}
						return ui.OneLine(gtx, l)
					})
				}),
				layout.Flexed(1, func(gtx C) D {
					return m.toggleTail.Layout(gtx, func(gtx C) D {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return layout.E.Layout(gtx, func(gtx C) D {
							hint := a.t("msg.show")
							switch {
							case m.Running:
								hint = a.t("msg.running")
							case m.Expanded:
								hint = a.t("msg.hide")
							}
							if m.HasDiff && !m.Running {
								_, stat := m.diff.view(th, m.DiffOld, m.DiffNew, a.t("diff.none"))
								hint = stat + " · " + hint
							}
							c := th.Muted
							if m.toggle.Hovered() || m.toggleTail.Hovered() {
								c = th.Text
							}
							return th.Label(12, hint, c, font.Normal).Layout(gtx)
						})
					})
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			if !m.Expanded || m.Running {
				return D{}
			}
			return layout.Inset{Top: 8, Left: 18}.Layout(gtx, func(gtx C) D {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return ui.Bordered(gtx, th.Code, th.Border, 8, func(gtx C) D {
					if m.HasDiff {
						w, _ := m.diff.view(th, m.DiffOld, m.DiffNew, a.t("diff.none"))
						return layout.UniformInset(10).Layout(gtx, w)
					}
					return layout.UniformInset(10).Layout(gtx, th.CodeLines(m.Output))
				})
			})
		}),
	)
}

// thinking shows the model's reasoning: streamed in full while running, then
// collapsed to a "Thought for Ns" line that can be expanded again.
func (a *App) thinking(gtx C, m *Message) D {
	th := a.th
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return m.toggle.Layout(gtx, func(gtx C) D {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						if m.Running {
							return ui.Dot(gtx, ui.Pulse(gtx, th.Accent), 8)
						}
						return ui.Ring(gtx, th.Muted, 9)
					}),
					ui.HSpace(10),
					layout.Rigid(func(gtx C) D {
						if m.Running {
							return th.Label(14, a.t("thinking.running"), ui.Pulse(gtx, th.Text), font.Medium).Layout(gtx)
						}
						c := th.Muted
						if m.toggle.Hovered() {
							c = th.Text
						}
						return th.Label(14, a.t("thinking.done", m.Secs), c, font.Medium).Layout(gtx)
					}),
					ui.HSpace(8),
					layout.Rigid(func(gtx C) D {
						if m.Running {
							return D{}
						}
						hint := a.t("msg.show")
						if m.Expanded {
							hint = a.t("msg.hide")
						}
						return th.Label(12, hint, th.Muted, font.Normal).Layout(gtx)
					}),
				)
			})
		}),
		layout.Rigid(func(gtx C) D {
			if !m.Expanded && !m.Running {
				return D{}
			}
			return layout.Inset{Top: 8, Left: 3}.Layout(gtx, func(gtx C) D {
				return ui.Ruled(gtx, th.BorderHi, func(gtx C) D {
					l := th.Label(14, m.Text, th.Muted, font.Normal)
					l.Font.Style = font.Italic
					l.LineHeightScale = 1.45
					return l.Layout(gtx)
				})
			})
		}),
	)
}

func (a *App) taskList(gtx C, m *Message) D {
	th := a.th
	done := 0
	for _, t := range m.Tasks {
		if t.Status == TaskDone {
			done++
		}
	}
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	return ui.Bordered(gtx, th.Surface, th.Border, 10, func(gtx C) D {
		return layout.Inset{Top: 10, Bottom: 10, Left: 14, Right: 14}.Layout(gtx, func(gtx C) D {
			children := []layout.FlexChild{
				layout.Rigid(func(gtx C) D {
					return layout.Flex{}.Layout(gtx,
						layout.Rigid(th.Label(12, a.t("tasks.title"), th.Muted, font.SemiBold).Layout),
						ui.HSpace(8),
						layout.Rigid(th.Label(12, itoa(done)+"/"+itoa(len(m.Tasks)), th.Muted, font.Normal).Layout),
					)
				}),
				ui.VSpace(6),
			}
			for _, t := range m.Tasks {
				children = append(children, layout.Rigid(func(gtx C) D {
					return layout.Inset{Top: 3, Bottom: 3}.Layout(gtx, func(gtx C) D {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								switch t.Status {
								case TaskDone:
									return ui.Dot(gtx, th.Success, 12)
								case TaskRunning:
									return ui.Dot(gtx, ui.Pulse(gtx, th.Accent), 12)
								}
								return ui.Ring(gtx, th.Muted, 12)
							}),
							ui.HSpace(10),
							layout.Rigid(func(gtx C) D {
								c, w := th.Text, font.Normal
								switch t.Status {
								case TaskDone:
									c = th.Muted
								case TaskRunning:
									w = font.Medium
								}
								return th.Label(14, t.Text, c, w).Layout(gtx)
							}),
						)
					})
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
	})
}
