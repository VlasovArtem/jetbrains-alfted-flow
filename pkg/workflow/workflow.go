package workflow

import (
	"fmt"
	"github.com/deanishe/awgo"
	"jetbrains-project-workflow/pkg/reader"
	"jetbrains-project-workflow/pkg/service"
	"time"
)

const (
	day   = 24 * time.Hour
	week  = 7 * day
	month = 4 * week
	year  = 12 * month
)

var (
	icons             = map[string]*aw.Icon{
		"Rider":        {Value: "icons/rider.png"},
		"DataGrip":     {Value: "icons/datagrip.png"},
		"IntellijIdea": {Value: "icons/idea.png"},
		"GoLand":       {Value: "icons/goland.png"},
	}
	ignoreProjects = map[string]string{
		"CodeWithMeGuest": "",
		"DataGrip":        "",
	}
)

func Open(wf *aw.Workflow) {
	query := wf.Args()[0]

	wf.Configure(aw.SuppressUIDs(true))

	projectService := service.New()

	if err := reader.ReadProjects(&projectService, ignoreProjects); err != nil {
		wf.FatalError(err)
	}

	if query == "" {
		for _, info := range projectService.GetProjects() {
			projectInfoToWfItem(&info, wf)
		}
	} else {
		for _, info := range projectService.FilterProjects(query) {
			projectInfoToWfItem(&info, wf)
		}
	}

	wf.WarnEmpty("No matching items", "Try a different query?")
	wf.SendFeedback()
}

func projectInfoToWfItem(projectInfo *service.ProjectInfo, wf *aw.Workflow) {
	if projectInfo.Valid {
		item := fmt.Sprintf("%s %s", projectInfo.Name, generateLastOpenDateString(projectInfo.ProjectOpenTimestamp))
		if projectInfo.Opened {
			item = item + " (Opened)"
		}
		wf.NewItem(item).
			Subtitle(projectInfo.Path).
			Icon(icons[projectInfo.Project]).
			Arg(projectInfo.JetbrainsAppPath, projectInfo.Path).
			Valid(projectInfo.Valid)
	}
}

func generateLastOpenDateString(projectOpenTimestamp int) string {
	now := time.Now()

	lastOpenTime := time.Unix(int64(projectOpenTimestamp), 0)

	duration := now.Sub(lastOpenTime)

	seconds := duration.Seconds()

	switch {
	case asYear(seconds) > 0:
		return fmt.Sprintf("%d year(s) ago", asYear(seconds))
	case asMonth(seconds) > 0:
		return fmt.Sprintf("%d month(s) ago", asMonth(seconds))
	case asWeek(seconds) > 0:
		return fmt.Sprintf("%d week(s) ago", asWeek(seconds))
	case asDay(seconds) > 0:
		return fmt.Sprintf("%d day(s) ago", asDay(seconds))
	case asHour(seconds) > 0:
		return fmt.Sprintf("%d hour(s) ago", asHour(seconds))
	case asMinute(seconds) > 0:
		return fmt.Sprintf("%d minute(s) ago", asMinute(seconds))
	default:
		return fmt.Sprintf("%d second(s) ago", int(seconds))
	}
}

func asYear(seconds float64) int {
	return int(seconds / year.Seconds())
}

func asMonth(seconds float64) int {
	return int(seconds / month.Seconds())
}

func asWeek(seconds float64) int {
	return int(seconds / week.Seconds())
}

func asDay(seconds float64) int {
	return int(seconds / day.Seconds())
}

func asHour(seconds float64) int {
	return int(seconds / time.Hour.Seconds())
}

func asMinute(seconds float64) int {
	return int(seconds / time.Minute.Seconds())
}
