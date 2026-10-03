package validator

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/nicolo-ribaudo/netex-validator/types"
)

const (
	unknownSeverityClass = "unknown"
)

// HTMLReporter generates professional HTML reports for validation results
type HTMLReporter struct {
	template *template.Template
}

// NewHTMLReporter creates a new HTML reporter
func NewHTMLReporter() *HTMLReporter {
	tmpl := template.Must(template.New("validation_report").Funcs(template.FuncMap{
		"severityClass": severityClass,
		"severityIcon":  severityIcon,
		"severityText":  severityText,
		"formatTime":    formatTime,
		"percentage":    percentage,
		"lower":         strings.ToLower,
	}).Parse(htmlTemplate))

	return &HTMLReporter{
		template: tmpl,
	}
}

// GenerateHTML generates an HTML report from validation results
func (r *HTMLReporter) GenerateHTML(result *ValidationResult) (string, error) {
	data := r.prepareTemplateData(result)

	var buf strings.Builder
	if err := r.template.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// prepareTemplateData prepares data for the HTML template
func (r *HTMLReporter) prepareTemplateData(result *ValidationResult) *HTMLTemplateData {
	summary := result.Summary()

	// Group issues by file
	issuesByFile := make(map[string][]ValidationReportEntry)
	for _, entry := range result.ValidationReportEntries {
		fileName := entry.FileName
		if fileName == "" {
			fileName = "Unknown"
		}
		issuesByFile[fileName] = append(issuesByFile[fileName], entry)
	}

	// Group issues by severity
	issuesBySeverity := make(map[string][]ValidationReportEntry)
	for _, entry := range result.ValidationReportEntries {
		severity := severityText(entry.Severity)
		issuesBySeverity[severity] = append(issuesBySeverity[severity], entry)
	}

	// Sort severity keys
	severityKeys := []string{"Critical", "Error", "Warning", "Info"}
	filteredSeverityKeys := []string{}
	for _, key := range severityKeys {
		if len(issuesBySeverity[key]) > 0 {
			filteredSeverityKeys = append(filteredSeverityKeys, key)
		}
	}

	// Group issues by rule
	issuesByRule := make(map[string][]ValidationReportEntry)
	for _, entry := range result.ValidationReportEntries {
		issuesByRule[entry.Name] = append(issuesByRule[entry.Name], entry)
	}

	// Calculate statistics
	stats := &ValidationStatistics{
		TotalIssues:      len(result.ValidationReportEntries),
		FilesProcessed:   result.FilesProcessed,
		ProcessingTime:   result.ProcessingTime,
		HasErrors:        !result.IsValid(),
		SeverityCounts:   make(map[string]int),
		SeverityPercents: make(map[string]float64),
	}

	totalIssues := len(result.ValidationReportEntries)
	for severity, issues := range issuesBySeverity {
		count := len(issues)
		stats.SeverityCounts[severity] = count
		if totalIssues > 0 {
			stats.SeverityPercents[severity] = float64(count) / float64(totalIssues) * 100
		}
	}

	return &HTMLTemplateData{
		Result:           result,
		Summary:          summary,
		Statistics:       stats,
		IssuesByFile:     issuesByFile,
		IssuesBySeverity: issuesBySeverity,
		IssuesByRule:     issuesByRule,
		SeverityKeys:     filteredSeverityKeys,
		GeneratedAt:      time.Now(),
	}
}

// HTMLTemplateData contains all data needed for HTML template
type HTMLTemplateData struct {
	Result           *ValidationResult
	Summary          ValidationSummary
	Statistics       *ValidationStatistics
	IssuesByFile     map[string][]ValidationReportEntry
	IssuesBySeverity map[string][]ValidationReportEntry
	IssuesByRule     map[string][]ValidationReportEntry
	SeverityKeys     []string
	GeneratedAt      time.Time
}

// ValidationStatistics contains statistical information about validation results
type ValidationStatistics struct {
	TotalIssues      int
	FilesProcessed   int
	ProcessingTime   time.Duration
	HasErrors        bool
	SeverityCounts   map[string]int
	SeverityPercents map[string]float64
}

// Template helper functions
func severityClass(severity types.Severity) string {
	switch severity {
	case types.CRITICAL:
		return "critical"
	case types.ERROR:
		return "error"
	case types.WARNING:
		return "warning"
	case types.INFO:
		return "info"
	default:
		return unknownSeverityClass
	}
}

func severityIcon(severity types.Severity) string {
	switch severity {
	case types.CRITICAL:
		return "⛔"
	case types.ERROR:
		return "❌"
	case types.WARNING:
		return "⚠️"
	case types.INFO:
		return "ℹ️"
	default:
		return "❓"
	}
}

func severityText(severity types.Severity) string {
	switch severity {
	case types.CRITICAL:
		return "Critical"
	case types.ERROR:
		return "Error"
	case types.WARNING:
		return "Warning"
	case types.INFO:
		return "Info"
	default:
		return "Unknown"
	}
}

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func percentage(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total) * 100
}

// HTML template for validation reports
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
    <title>NetEX Validation Report - {{.Result.ValidationReportID}}</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            line-height: 1.6;
            color: #333;
            background-color: #f5f5f5;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }

        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
            border-radius: 10px;
            margin-bottom: 30px;
            text-align: center;
        }

        .header h1 {
            font-size: 2.5em;
            margin-bottom: 10px;
        }

        .header .subtitle {
            font-size: 1.2em;
            opacity: 0.9;
        }

        .summary-cards {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }

        .summary-card {
            background: white;
            padding: 25px;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            text-align: center;
        }

        .summary-card h3 {
            font-size: 2.5em;
            margin-bottom: 10px;
        }

        .summary-card.total { border-left: 5px solid #6c7ce7; }
        .summary-card.files { border-left: 5px solid #51cf66; }
        .summary-card.time { border-left: 5px solid #ff8cc8; }
        .summary-card.status { border-left: 5px solid #ffa502; }

        .tabs {
            background: white;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            overflow: hidden;
        }

        .tab-buttons {
            display: flex;
            background: #f8f9fb;
            border-bottom: 1px solid #e1e5e9;
        }

        .tab-button {
            flex: 1;
            padding: 15px 20px;
            border: none;
            background: none;
            cursor: pointer;
            font-size: 16px;
            font-weight: 500;
            color: #666;
            transition: all 0.3s ease;
        }

        .tab-button:hover {
            background: #e9ecef;
            color: #333;
        }

        .tab-button.active {
            background: white;
            color: #667eea;
            border-bottom: 2px solid #667eea;
        }

        .tab-content {
            display: none;
            padding: 30px;
        }

        .tab-content.active {
            display: block;
        }

        .issue-list {
            list-style: none;
        }

        .issue-item {
            background: #f8f9fa;
            margin-bottom: 15px;
            padding: 20px;
            border-radius: 8px;
            border-left: 4px solid #ddd;
        }

        .issue-item.critical { border-left-color: #dc3545; }
        .issue-item.error { border-left-color: #fd7e14; }
        .issue-item.warning { border-left-color: #ffc107; }
        .issue-item.info { border-left-color: #17a2b8; }

        .issue-header {
            display: flex;
            align-items: center;
            margin-bottom: 10px;
        }

        .severity-badge {
            padding: 4px 12px;
            border-radius: 20px;
            font-size: 12px;
            font-weight: 600;
            text-transform: uppercase;
            margin-right: 15px;
        }

        .severity-badge.critical {
            background: #dc3545;
            color: white;
        }

        .severity-badge.error {
            background: #fd7e14;
            color: white;
        }

        .severity-badge.warning {
            background: #ffc107;
            color: #333;
        }

        .severity-badge.info {
            background: #17a2b8;
            color: white;
        }

        .issue-title {
            font-size: 18px;
            font-weight: 600;
            color: #333;
            flex: 1;
        }

        .issue-details {
            color: #666;
            font-size: 14px;
            line-height: 1.5;
        }

        .issue-meta {
            margin-top: 10px;
            padding-top: 10px;
            border-top: 1px solid #e9ecef;
            font-size: 12px;
            color: #888;
        }

        .file-group {
            margin-bottom: 30px;
        }

        .file-group h3 {
            color: #667eea;
            margin-bottom: 15px;
            padding-bottom: 5px;
            border-bottom: 2px solid #667eea;
            font-size: 20px;
        }

        .severity-stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
            margin-bottom: 25px;
        }

        .severity-stat {
            background: #f8f9fa;
            padding: 15px;
            border-radius: 8px;
            text-align: center;
        }

        .severity-stat .count {
            font-size: 24px;
            font-weight: 600;
            margin-bottom: 5px;
        }

        .severity-stat .percentage {
            font-size: 12px;
            color: #666;
        }

        .progress-bar {
            width: 100%;
            height: 8px;
            background: #e9ecef;
            border-radius: 4px;
            overflow: hidden;
            margin-top: 5px;
        }

        .progress-fill {
            height: 100%;
            transition: width 0.3s ease;
        }

        .progress-fill.critical { background: #dc3545; }
        .progress-fill.error { background: #fd7e14; }
        .progress-fill.warning { background: #ffc107; }
        .progress-fill.info { background: #17a2b8; }

        .footer {
            text-align: center;
            margin-top: 40px;
            padding: 20px;
            color: #666;
            font-size: 14px;
        }

        @media (max-width: 768px) {
            .container {
                padding: 10px;
            }

            .header h1 {
                font-size: 2em;
            }

            .tab-buttons {
                flex-direction: column;
            }

            .summary-cards {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>NetEX Validation Report</h1>
            <div class="subtitle">{{.Result.ValidationReportID}} - {{formatTime .GeneratedAt}}</div>
        </div>

        <div class="summary-cards">
            <div class="summary-card total">
                <h3>{{.Statistics.TotalIssues}}</h3>
                <p>Total Issues</p>
            </div>
            <div class="summary-card files">
                <h3>{{.Statistics.FilesProcessed}}</h3>
                <p>Files Processed</p>
            </div>
            <div class="summary-card time">
                <h3>{{printf "%.2f" .Statistics.ProcessingTime.Seconds}}s</h3>
                <p>Processing Time</p>
            </div>
            <div class="summary-card status">
                <h3>{{if .Statistics.HasErrors}}❌{{else}}✅{{end}}</h3>
                <p>{{if .Statistics.HasErrors}}Failed{{else}}Passed{{end}}</p>
            </div>
        </div>

        <div class="tabs">
            <div class="tab-buttons">
                <button class="tab-button active" onclick="showTab('all')">All Issues</button>
                <button class="tab-button" onclick="showTab('file')">By File</button>
                <button class="tab-button" onclick="showTab('severity')">By Severity</button>
                <button class="tab-button" onclick="showTab('rule')">By Rule</button>
            </div>

            <div id="all" class="tab-content active">
                <h2>All Validation Issues ({{.Statistics.TotalIssues}})</h2>
                <ul class="issue-list">
                    {{range .Result.ValidationReportEntries}}
                    <li class="issue-item {{severityClass .Severity}}">
                        <div class="issue-header">
                            <span class="severity-badge {{severityClass .Severity}}">
                                {{severityIcon .Severity}} {{severityText .Severity}}
                            </span>
                            <span class="issue-title">{{.Name}}</span>
                        </div>
                        <div class="issue-details">{{.Message}}</div>
                        <div class="issue-meta">
                            File: {{.FileName}} 
                            {{if .Location.ElementID}}| Element: {{.Location.ElementID}}{{end}}
                            {{if .Location.XPath}}| XPath: {{.Location.XPath}}{{end}}
                        </div>
                    </li>
                    {{end}}
                </ul>
            </div>

            <div id="file" class="tab-content">
                <h2>Issues by File</h2>
                {{range $fileName, $issues := .IssuesByFile}}
                <div class="file-group">
                    <h3>{{$fileName}} ({{len $issues}} issues)</h3>
                    <ul class="issue-list">
                        {{range $issues}}
                        <li class="issue-item {{severityClass .Severity}}">
                            <div class="issue-header">
                                <span class="severity-badge {{severityClass .Severity}}">
                                    {{severityIcon .Severity}} {{severityText .Severity}}
                                </span>
                                <span class="issue-title">{{.Name}}</span>
                            </div>
                            <div class="issue-details">{{.Message}}</div>
                            {{if .Location.ElementID}}<div class="issue-meta">Element: {{.Location.ElementID}}</div>{{end}}
                        </li>
                        {{end}}
                    </ul>
                </div>
                {{end}}
            </div>

            <div id="severity" class="tab-content">
                <h2>Issues by Severity</h2>
                <div class="severity-stats">
                    {{range .SeverityKeys}}
                    {{$count := index $.Statistics.SeverityCounts .}}
                    {{$percent := index $.Statistics.SeverityPercents .}}
                    <div class="severity-stat">
                        <div class="count">{{$count}}</div>
                        <div>{{.}}</div>
                        <div class="percentage">{{printf "%.1f" $percent}}%</div>
                        <div class="progress-bar">
                            <div class="progress-fill {{. | lower}}" style="width: {{$percent}}%"></div>
                        </div>
                    </div>
                    {{end}}
                </div>

                {{range .SeverityKeys}}
                {{$issues := index $.IssuesBySeverity .}}
                <div class="file-group">
                    <h3>{{.}} Issues ({{len $issues}})</h3>
                    <ul class="issue-list">
                        {{range $issues}}
                        <li class="issue-item {{severityClass .Severity}}">
                            <div class="issue-header">
                                <span class="issue-title">{{.Name}}</span>
                            </div>
                            <div class="issue-details">{{.Message}}</div>
                            <div class="issue-meta">File: {{.FileName}}</div>
                        </li>
                        {{end}}
                    </ul>
                </div>
                {{end}}
            </div>

            <div id="rule" class="tab-content">
                <h2>Issues by Validation Rule</h2>
                {{range $ruleName, $issues := .IssuesByRule}}
                <div class="file-group">
                    <h3>{{$ruleName}} ({{len $issues}} issues)</h3>
                    <ul class="issue-list">
                        {{range $issues}}
                        <li class="issue-item {{severityClass .Severity}}">
                            <div class="issue-header">
                                <span class="severity-badge {{severityClass .Severity}}">
                                    {{severityIcon .Severity}} {{severityText .Severity}}
                                </span>
                            </div>
                            <div class="issue-details">{{.Message}}</div>
                            <div class="issue-meta">File: {{.FileName}}</div>
                        </li>
                        {{end}}
                    </ul>
                </div>
                {{end}}
            </div>
        </div>

        <div class="footer">
            <p>Generated by NetEX Validator Library at {{formatTime .GeneratedAt}}</p>
            <p>Report ID: {{.Result.ValidationReportID}} | Codespace: {{.Result.Codespace}}</p>
        </div>
    </div>

    <script>
        function showTab(tabName) {
            // Hide all tab contents
            const contents = document.querySelectorAll('.tab-content');
            contents.forEach(content => content.classList.remove('active'));

            // Remove active class from all buttons
            const buttons = document.querySelectorAll('.tab-button');
            buttons.forEach(button => button.classList.remove('active'));

            // Show selected tab content
            document.getElementById(tabName).classList.add('active');

            // Add active class to clicked button
            event.target.classList.add('active');
        }
    </script>
</body>
</html>`
