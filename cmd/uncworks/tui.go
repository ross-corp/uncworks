// tui.go — uncworks tui: Bubble Tea terminal UI for UNCWORKS.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"connectrpc.com/connect"

	apiv1 "github.com/uncworks/aot/gen/go/api/v1"
	apiv1connect "github.com/uncworks/aot/gen/go/api/v1/apiv1connect"
)

// ── Styles ────────────────────────────────────────────────────────────────────

var (
	styleTitle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62"))
	styleStatus    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleSelected  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	styleError     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleHelp      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	styleBorder    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
)

// ── View types ────────────────────────────────────────────────────────────────

type tuiView int

const (
	viewRunList tuiView = iota
	viewLog
	viewSubmit
	viewHelp
)

// ── Messages ──────────────────────────────────────────────────────────────────

type runsLoadedMsg struct{ runs []*apiv1.AgentRun }
type runsErrMsg struct{ err error }
type logLineMsg struct{ line string }
type logDoneMsg struct{}
type submitDoneMsg struct{ runID string }
type submitErrMsg struct{ err error }
type tickMsg struct{}

// ── Run list item ─────────────────────────────────────────────────────────────

type runItem struct{ run *apiv1.AgentRun }

func (r runItem) Title() string {
	phase := "?"
	if r.run.GetStatus() != nil {
		phase = phaseLabel(r.run.GetStatus().GetPhase())
	}
	dur := runDuration(r.run)
	if dur != "-" {
		return fmt.Sprintf("[%s] %s (%s)", phase, r.run.GetName(), dur)
	}
	return fmt.Sprintf("[%s] %s", phase, r.run.GetName())
}

func (r runItem) Description() string {
	var parts []string
	if dn := r.run.GetSpec().GetDisplayName(); dn != "" {
		parts = append(parts, dn)
	}
	if p := r.run.GetSpec().GetProject(); p != "" {
		parts = append(parts, p)
	}
	if r.run.GetStatus() != nil && r.run.GetStatus().GetMessage() != "" {
		parts = append(parts, r.run.GetStatus().GetMessage())
	}
	if len(parts) > 0 {
		return strings.Join(parts, " · ")
	}
	return r.run.GetId()
}
func (r runItem) FilterValue() string { return r.run.GetName() }

func phaseLabel(p apiv1.AgentRunPhase) string {
	switch p {
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_PENDING:
		return "PENDING"
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_RUNNING:
		return "RUNNING"
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_WAITING_FOR_INPUT:
		return "WAITING"
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_SUCCEEDED:
		return "DONE"
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_FAILED:
		return "FAILED"
	case apiv1.AgentRunPhase_AGENT_RUN_PHASE_CANCELLED:
		return "CANCELLED"
	default:
		return "UNKNOWN"
	}
}

func runDuration(r *apiv1.AgentRun) string {
	if r.GetStatus().GetStartedAt() == nil {
		return "-"
	}
	start := r.GetStatus().GetStartedAt().AsTime()
	var end time.Time
	if r.GetStatus().GetCompletedAt() != nil {
		end = r.GetStatus().GetCompletedAt().AsTime()
	} else {
		end = time.Now()
	}
	return formatDuration(end.Sub(start))
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return "<1s"
	}
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", d/time.Second)
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%02ds", d/time.Minute, (d%time.Minute)/time.Second)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh%02dm", d/time.Hour, (d%time.Hour)/time.Minute)
	}
	return fmt.Sprintf("%dd%02dh", d/(24*time.Hour), (d%(24*time.Hour))/time.Hour)
}

// ── Model ─────────────────────────────────────────────────────────────────────

type tuiModel struct {
	client    apiv1connect.AOTServiceClient
	view      tuiView
	width     int
	height    int
	err       error

	// run list
	list    list.Model
	spinner spinner.Model
	loading bool

	// log view
	logPort    viewport.Model
	logLines   []string
	watchRunID string
	logDone    bool

	// submit form
	repoInput   textinput.Model
	branchInput textinput.Model
	specInput   textarea.Model
	submitFocus int // 0=repo, 1=branch, 2=spec, 3=submit button

	// help
	showHelp bool
}

func newTUIModel(client apiv1connect.AOTServiceClient) tuiModel {
	// List
	delegate := list.NewDefaultDelegate()
	l := list.New(nil, delegate, 0, 0)
	l.Title = "UNCWORKS — Agent Runs"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	// Spinner
	s := spinner.New()
	s.Spinner = spinner.Dot

	// Viewport for logs
	vp := viewport.New(0, 0)
	vp.SetContent("")

	// Submit form inputs
	repo := textinput.New()
	repo.Placeholder = "https://github.com/owner/repo"
	repo.Focus()

	branch := textinput.New()
	branch.Placeholder = "main"

	spec := textarea.New()
	spec.Placeholder = "Describe what you want the agent to do..."
	spec.SetHeight(8)

	return tuiModel{
		client:      client,
		view:        viewRunList,
		list:        l,
		spinner:     s,
		loading:     true,
		logPort:     vp,
		repoInput:   repo,
		branchInput: branch,
		specInput:   spec,
	}
}

func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadRuns(m.client), tickEvery(10*time.Second))
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetSize(msg.Width, msg.Height-2)
		m.logPort.Width = msg.Width
		m.logPort.Height = msg.Height - 4
		m.specInput.SetWidth(msg.Width - 4)
		return m, nil

	case runsLoadedMsg:
		m.loading = false
		items := make([]list.Item, len(msg.runs))
		for i, r := range msg.runs {
			items[i] = runItem{run: r}
		}
		m.list.SetItems(items)
		return m, nil

	case runsErrMsg:
		m.loading = false
		m.err = msg.err
		return m, nil

	case logLineMsg:
		m.logLines = append(m.logLines, msg.line)
		m.logPort.SetContent(strings.Join(m.logLines, "\n"))
		if m.logPort.AtBottom() {
			m.logPort.GotoBottom()
		}
		return m, nil

	case logDoneMsg:
		m.logDone = true
		return m, nil

	case submitDoneMsg:
		m.view = viewRunList
		m.loading = true
		return m, loadRuns(m.client)

	case submitErrMsg:
		m.err = msg.err
		return m, nil

	case tickMsg:
		if m.view == viewRunList {
			return m, tea.Batch(loadRuns(m.client), tickEvery(10*time.Second))
		}
		return m, tickEvery(10 * time.Second)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Delegate to active view component.
	var cmd tea.Cmd
	switch m.view {
	case viewRunList:
		m.list, cmd = m.list.Update(msg)
	case viewLog:
		m.logPort, cmd = m.logPort.Update(msg)
	}
	return m, cmd
}

func (m tuiModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.view {
	case viewRunList:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "?":
			m.view = viewHelp
			return m, nil
		case "n":
			m.view = viewSubmit
			m.submitFocus = 0
			m.repoInput.Focus()
			m.branchInput.Blur()
			return m, nil
		case "enter":
			if item, ok := m.list.SelectedItem().(runItem); ok {
				m.view = viewLog
				m.logLines = nil
				m.logDone = false
				m.watchRunID = item.run.GetId()
				m.logPort.SetContent("")
				return m, streamLogs(m.client, item.run.GetId())
			}
		}
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd

	case viewLog:
		switch msg.String() {
		case "q", "esc":
			m.view = viewRunList
			return m, nil
		}
		var cmd tea.Cmd
		m.logPort, cmd = m.logPort.Update(msg)
		return m, cmd

	case viewSubmit:
		return m.handleSubmitKey(msg)

	case viewHelp:
		m.view = viewRunList
		return m, nil
	}
	return m, nil
}

func (m tuiModel) handleSubmitKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.view = viewRunList
		return m, nil
	case "tab":
		m.submitFocus = (m.submitFocus + 1) % 4
		m.repoInput.Blur()
		m.branchInput.Blur()
		m.specInput.Blur()
		switch m.submitFocus {
		case 0:
			m.repoInput.Focus()
		case 1:
			m.branchInput.Focus()
		case 2:
			_ = m.specInput.Focus()
		}
		return m, nil
	case "ctrl+s", "enter":
		if m.submitFocus == 3 {
			return m, m.doSubmit()
		}
	}
	var cmd tea.Cmd
	switch m.submitFocus {
	case 0:
		m.repoInput, cmd = m.repoInput.Update(msg)
	case 1:
		m.branchInput, cmd = m.branchInput.Update(msg)
	case 2:
		m.specInput, cmd = m.specInput.Update(msg)
	}
	return m, cmd
}

func (m tuiModel) doSubmit() tea.Cmd {
	repoURL := m.repoInput.Value()
	branch := m.branchInput.Value()
	if branch == "" {
		branch = "main"
	}
	prompt := m.specInput.Value()
	client := m.client
	return func() tea.Msg {
		req := connect.NewRequest(&apiv1.CreateAgentRunRequest{
			Spec: &apiv1.AgentRunSpec{
				Repos:  []*apiv1.Repository{{Url: repoURL, Branch: branch}},
				Prompt: prompt,
			},
		})
		resp, err := client.CreateAgentRun(context.Background(), req)
		if err != nil {
			return submitErrMsg{err: err}
		}
		run := resp.Msg.GetAgentRun()
		runID := ""
		if run != nil {
			runID = run.GetId()
		}
		return submitDoneMsg{runID: runID}
	}
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m tuiModel) View() string {
	if m.err != nil {
		msg := humanizeErr(m.err)
		return styleError.Render(fmt.Sprintf("Error: %s\n\nRun 'uncworks open' to start the local server, or 'uncworks connect <addr>' to set a remote address.\nPress q to quit.", msg))
	}

	switch m.view {
	case viewRunList:
		if m.loading {
			return fmt.Sprintf("\n  %s Loading runs...\n", m.spinner.View())
		}
		return m.list.View() + "\n" + styleHelp.Render("  n new  enter view logs  ? help  q quit")

	case viewLog:
		header := styleTitle.Render(fmt.Sprintf("Logs: %s", m.watchRunID))
		status := ""
		if m.logDone {
			status = styleStatus.Render(" [stream ended]")
		}
		footer := styleHelp.Render("  ↑/↓ scroll  q/esc back")
		return header + status + "\n" + m.logPort.View() + "\n" + footer

	case viewSubmit:
		var sb strings.Builder
		sb.WriteString(styleTitle.Render("Submit New Agent Run") + "\n\n")
		sb.WriteString(fmt.Sprintf("  Repository URL:  %s\n", m.repoInput.View()))
		sb.WriteString(fmt.Sprintf("  Branch:          %s\n", m.branchInput.View()))
		sb.WriteString(fmt.Sprintf("  Prompt:\n%s\n", m.specInput.View()))
		submitBtn := "  [ Submit ]"
		if m.submitFocus == 3 {
			submitBtn = styleSelected.Render("  [ Submit ]")
		}
		sb.WriteString(submitBtn + "\n\n")
		sb.WriteString(styleHelp.Render("  tab next field  ctrl+s submit  esc cancel"))
		return sb.String()

	case viewHelp:
		return styleBorder.Render(
			styleTitle.Render("Key Bindings") + "\n\n" +
				"  Run list:\n" +
				"    j/k or ↑/↓  Navigate\n" +
				"    enter        View logs\n" +
				"    n            New run\n" +
				"    /            Filter\n" +
				"    q            Quit\n\n" +
				"  Log view:\n" +
				"    j/k or ↑/↓  Scroll\n" +
				"    q/esc        Back\n\n" +
				"  Submit form:\n" +
				"    tab          Next field\n" +
				"    ctrl+s       Submit\n" +
				"    esc          Cancel\n\n" +
				styleHelp.Render("Press any key to close"),
		)
	}
	return ""
}

// ── Commands ──────────────────────────────────────────────────────────────────

func loadRuns(client apiv1connect.AOTServiceClient) tea.Cmd {
	return func() tea.Msg {
		req := connect.NewRequest(&apiv1.ListAgentRunsRequest{Limit: 50})
		resp, err := client.ListAgentRuns(context.Background(), req)
		if err != nil {
			return runsErrMsg{err: err}
		}
		return runsLoadedMsg{runs: resp.Msg.GetAgentRuns()}
	}
}

func streamLogs(client apiv1connect.AOTServiceClient, runID string) tea.Cmd {
	return func() tea.Msg {
		req := connect.NewRequest(&apiv1.WatchAgentRunRequest{Id: runID})
		stream, err := client.WatchAgentRun(context.Background(), req)
		if err != nil {
			return logLineMsg{line: fmt.Sprintf("error: %v", err)}
		}
		// Return the first line synchronously; subsequent lines are pushed via a goroutine.
		// For simplicity in v1, stream all into logLines via a blocking command chain.
		var lines []string
		for stream.Receive() {
			ev := stream.Msg()
			if ev.GetPayload() != "" {
				lines = append(lines, ev.GetPayload())
			}
		}
		if err := stream.Err(); err != nil && err != io.EOF {
			lines = append(lines, fmt.Sprintf("[stream error: %v]", err))
		}
		return logLineMsg{line: strings.Join(lines, "\n")}
	}
}

func tickEvery(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// humanizeErr converts a connect-rpc or network error into a short, readable message.
func humanizeErr(err error) string {
	if err == nil {
		return ""
	}
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		switch connectErr.Code() {
		case connect.CodeUnavailable:
			return "server unavailable — is UNCWORKS running?"
		case connect.CodeUnauthenticated:
			return "authentication required — check your credentials"
		case connect.CodePermissionDenied:
			return "permission denied"
		case connect.CodeNotFound:
			return "resource not found"
		case connect.CodeUnimplemented:
			return "feature not available in this server version"
		case connect.CodeResourceExhausted:
			return fmt.Sprintf("request rejected: %s", connectErr.Message())
		default:
			return connectErr.Message()
		}
	}
	msg := err.Error()
	// connection refused is the most common failure for local setups
	if strings.Contains(msg, "connection refused") {
		return "connection refused — is 'uncworks open' running?"
	}
	if strings.Contains(msg, "no such host") {
		return fmt.Sprintf("host not found — check 'uncworks connect' address")
	}
	return msg
}

// ── Entry point ───────────────────────────────────────────────────────────────

func runTUI(args []string) error {
	fs := flag.NewFlagSet("tui", flag.ContinueOnError)
	serverAddr := fs.String("server", "", "gRPC server address (overrides config)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: uncworks tui [flags]\n\nLaunch the terminal UI.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	addr := *serverAddr
	if addr == "" {
		cfg, err := loadConfig()
		if err == nil && cfg.Server.Address != "" {
			addr = cfg.Server.Address
		} else {
			addr = fmt.Sprintf("http://localhost:%d", 50055)
		}
	}

	// Ensure addr has scheme for connect-go HTTP client.
	if !strings.HasPrefix(addr, "http") {
		addr = "http://" + addr
	}

	client := apiv1connect.NewAOTServiceClient(http.DefaultClient, addr)
	model := newTUIModel(client)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
