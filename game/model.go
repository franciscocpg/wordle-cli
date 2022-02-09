/* -----------------------------------------------------------------------------
 * Copyright (c) Nimble Bun Works. All rights reserved.
 * This software is licensed under the MIT license.
 * See the LICENSE file for further information.
 * -------------------------------------------------------------------------- */

package game

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"pkg.nimblebun.works/clipboard"
	"pkg.nimblebun.works/wordle-cli/common"
	"pkg.nimblebun.works/wordle-cli/common/save"
)

type AppModel struct {
	ID                int
	Word              [5]byte
	LetterStates      map[byte]common.LetterState
	Grid              [common.WordleMaxGuesses][common.WordleWordLength]*common.GridItem
	CurrentRow        int
	CurrentColumn     int
	GameType          common.GameType
	GameState         common.GameState
	WordState         common.WordState
	SaveData          *save.SaveFile
	NewGame           bool
	DisplayStatistics bool
}

const dialogBoxWidth = 23

var (
	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#ff0000")).
			Width(dialogBoxWidth).
			MarginLeft(1).
			BorderTop(true).
			BorderLeft(true).
			BorderRight(true).
			BorderBottom(true)

	emptyBoxStyle = lipgloss.NewStyle().
			Padding(2, 0)

	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF7DB")).
			Background(lipgloss.Color("#888B7E")).
			Padding(0, 3).
			MarginTop(1).
			Underline(true)
)

func NewGame(word string, gameType common.GameType, id int) *AppModel {
	model := &AppModel{}

	if gameType != common.GameTypeRandom {
		var saveData *save.SaveFile
		saveData, err := save.Load(gameType.ID())
		if err != nil {
			saveData = save.New()
		}

		model.SaveData = saveData
	}

	splitWord := strings.Split(word, "")
	for i, letter := range splitWord {
		model.Word[i] = letter[0]
	}

	model.ID = id
	model.GameType = gameType
	model.GameState = common.GameStateRunning
	model.WordState = common.WordStateOk
	model.LetterStates = make(map[byte]common.LetterState, 26)
	model.CurrentRow = 0
	model.CurrentColumn = 0
	model.NewGame = true
	model.DisplayStatistics = false

	if gameType != common.GameTypeRandom && model.SaveData.LastGameID == model.ID {
		model.GameState = model.SaveData.LastGameStatus
		model.NewGame = false

		for i := range model.SaveData.LastGameGrid {
			for j := range model.SaveData.LastGameGrid[i] {
				item := model.SaveData.LastGameGrid[i][j]

				if item != nil {
					model.setGridItem(i, j, item.Letter, item.State)

					model.CurrentColumn = j + 1
					model.CurrentRow = i + 1
				}
			}
		}
	}

	return model
}

func (m AppModel) Init() tea.Cmd {
	return nil
}

func (m *AppModel) View() string {
	grid := m.renderGrid()

	if m.GameState != common.GameStateRunning {
		_ = clipboard.WriteAll(m.getShareString())

		if m.NewGame && m.GameType != common.GameTypeRandom {
			m.save()
			m.NewGame = false
		}

		var finalBlock string

		if m.DisplayStatistics {
			finalBlock = m.renderStatisticsBlock()
		} else {
			finalBlock = m.renderFinalMessageBlock()
		}

		return lipgloss.JoinHorizontal(lipgloss.Top, grid, finalBlock)
	}

	keyboard := m.renderKeyboard()

	trailing := lipgloss.NewStyle().Padding(2, 0).Render(m.renderTrailingBlock())

	getMessageDialog := func() string {
		switch m.WordState {
		case common.WordStateNotEnoughLetters,
			common.WordStateNotInList:
			getMessage := func() string {
				if m.WordState == common.WordStateNotInList {
					return "Not in word list"
				}

				return "Not enough letters"
			}

			message := lipgloss.NewStyle().Width(dialogBoxWidth).Align(lipgloss.Center).Render(getMessage())
			okButton := buttonStyle.Render("Ok")
			buttons := lipgloss.JoinHorizontal(lipgloss.Top, okButton)
			ui := lipgloss.JoinVertical(lipgloss.Center, message, buttons)

			return dialogBoxStyle.Render(ui)
		}

		return emptyBoxStyle.String()
	}

	game := lipgloss.JoinHorizontal(lipgloss.Top, grid, keyboard)
	return lipgloss.JoinVertical(lipgloss.Left, getMessageDialog(), game, trailing)
}

func (m *AppModel) handleKeyDown(t tea.KeyType, r []rune) (tea.Model, tea.Cmd) {
	switch t {
	case tea.KeyBackspace:
		return m, m.backspace()
	case tea.KeyCtrlC:
		return m, m.quit()
	case tea.KeyEnter:
		return m, m.enter()
	case tea.KeyRight:
		return m, m.displayStatistics()
	case tea.KeyLeft:
		return m, m.displayGameSummary()
	case tea.KeyCtrlN:
		return m, m.new()
	case tea.KeyRunes:
		if len(r) != 1 {
			return m, nil
		}

		return m, m.input(r[0])
	default:
		return m, nil
	}
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyDown(msg.Type, msg.Runes)
	default:
		return m, nil
	}
}
