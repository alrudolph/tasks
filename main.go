package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/alrudolph/tasks/linear-api"
	"github.com/atotto/clipboard"
	"github.com/jroimartin/gocui"
)

var items = [][]string{}
var cursor int = 1
var offset int

const (
	boldStart = "\033[1m" // ANSI escape code for bold text
	boldEnd   = "\033[0m" // Reset formatting
)

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("list", 0, 0, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Active Issues"
		v.Highlight = true
		v.SelFgColor = gocui.ColorMagenta
		v.Wrap = false
		renderList(v, maxY-2) // Adjust for title bar

		err := v.SetCursor(0, cursorPosition())

		if err != nil {
			return err
		}
	}
	return nil
}

func renderList(v *gocui.View, maxVisible int) {
	v.Clear()
	visibleItems := min(maxVisible, len(items)-1)

	formatted := formatTable(items)

	// Pinned first item (always at the top)
	fmt.Fprintf(v, "  %s%s%s\n", boldStart, formatted[0], boldEnd)

	// Render scrollable items (excluding pinned)
	for i := offset + 1; i < offset+visibleItems+1; i++ {
		fmt.Fprintln(v, " ", formatted[i])
	}
}

func moveCursor(g *gocui.Gui, dy int) error {
	v, err := g.View("list")
	if err != nil {
		return err
	}
	_, maxY := g.Size()
	maxVisible := maxY - 3 // Excluding the title bar

	cursor += dy

	// Ensure the cursor does not go to the pinned item (index 0)
	if cursor < 1 {
		cursor = 1
	} else if cursor >= len(items) {
		cursor = len(items) - 1
	}

	// Adjust scrolling (excluding the pinned first item)
	if cursor < 1+offset {
		offset = cursor - 1
	} else if cursor >= 1+offset+maxVisible {
		offset = cursor - maxVisible
	}

	renderList(v, maxVisible)

	err = v.SetCursor(0, cursorPosition())

	if err != nil {
		return err
	}

	err = v.SetOrigin(0, 0)

	if err != nil {
		return err
	}

	return nil
}

// Cursor position in the view (accounting for pinned item)
func cursorPosition() int {
	return cursor - offset
}

func copyToClipboard(g *gocui.Gui, v *gocui.View) error {
	selectedItem := items[cursor]
	issueId := selectedItem[1]
	if err := clipboard.WriteAll(issueId); err != nil {
		return err
	}

	v, err := g.View("list")
	if err != nil {
		return nil
	}

	g.Update(func(g *gocui.Gui) error {
		v.Title = "Copied: " + issueId
		return nil
	})
	return nil
}

func openInBrowserHandler(g *gocui.Gui, v *gocui.View) error {
	selectedItem := items[cursor]
	issueId := selectedItem[1]
	url := fmt.Sprintf("https://linear.app/%s/issue/%s", "", issueId)

	if err := openInBrowser(url); err != nil {
		return err
	}

	v, err := g.View("list")
	if err != nil {
		return nil
	}

	g.Update(func(g *gocui.Gui) error {
		v.Title = "Opened in browser: " + url
		return nil
	})
	return nil
}

func moveUp(g *gocui.Gui, v *gocui.View) error {
	return moveCursor(g, -1)
}

func moveDown(g *gocui.Gui, v *gocui.View) error {
	return moveCursor(g, 1)
}

func keybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("", 'w', gocui.ModNone, moveUp); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone, moveUp); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 's', gocui.ModNone, moveDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone, moveDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'c', gocui.ModNone, copyToClipboard); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'W', gocui.ModNone, openInBrowserHandler); err != nil {
		return err
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func main() {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil || g == nil {
		log.Fatal("Failed to initialize gocui")
	}
	defer g.Close()

	getLinearData()

	g.SetManagerFunc(layout)

	if err := keybindings(g); err != nil {
		log.Fatal(err)
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatal(err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getLinearData() {

	var first int32 = 100

	// name := ""
	stateName := "Active"
	teamName := ""

	filter := linear.IssueFilter{
		// Assignee: &linear.NullableUserFilter{
		// 	Name: &linear.StringComparator{Eq: &name},
		// },
		State: &linear.WorkflowStateFilter{
			Name: &linear.StringComparator{Eq: &stateName},
		},
		Team: &linear.TeamFilter{
			Name: &linear.StringComparator{Eq: &teamName},
		},
	}

	// res, err := linear.FetchMe(
	// 	linear.DefaultClient(),
	// 	context.Background(),
	// )

	res, err := linear.FetchIssues(
		linear.DefaultClient(),
		context.Background(),
		&filter,
		&first,
		nil,
	)

	// res, err := linear.FetchTeams(
	// 	linear.DefaultClient(),
	// 	context.Background(),
	// )

	if err != nil {
		log.Fatal(err)
	}

	if res == nil {
		log.Fatal("res is nil")
	}

	// for _, team := range res.Nodes {
	// 	fmt.Println("team:", team.Name)
	// }

	items = [][]string{
		{strconv.Itoa(len(res.Nodes)), "Identifier", "Status", "Title", "Assignee"},
	}

	for i, issue := range res.Nodes {
		items = append(items, []string{
			strconv.Itoa(i + 1),
			issue.Identifier,
			issue.State.Name,
			issue.Title,
			issue.Assignee.Name,
		})
		// fmt.Printf("%s [%s] %s (%s)\n", issue.Identifier, issue.State.Name, issue.Title, issue.Assignee.Name)
	}

	// for _, row := range formatTable(data) {
	// 	fmt.Println(row)
	// }

	// fmt.Println("res:", res)
}

func openInBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/C", "start", url)
	case "linux":
		// Open URL in Windows default browser from WSL
		cmd = exec.Command("cmd.exe", "/C", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		return fmt.Errorf("unsupported OS")
	}
	return cmd.Start()
}

func formatTable(table [][]string) []string {
	if len(table) == 0 {
		return nil
	}

	// Determine the maximum width of each column
	colWidths := make([]int, len(table[0]))
	for _, row := range table {
		for i, col := range row {
			if len(col) > colWidths[i] {
				colWidths[i] = len(col)
			}
		}
	}

	// Create formatted rows
	var result []string
	for _, row := range table {
		var line strings.Builder
		for i, col := range row {
			line.WriteString(col)
			if i < len(row)-1 { // Add padding except for the last column
				line.WriteString(strings.Repeat(" ", colWidths[i]-len(col)+2))
			}
		}
		result = append(result, line.String())
	}

	return result
}
