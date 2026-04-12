package tools

import (
	"strconv"
	"strings"
)

type plannedPPTXPresentation struct {
	ThemeName string
	Slides    []plannedPPTXSlide
}

type plannedPPTXSlide struct {
	Slide  pptxSlide
	Layout string
}

func planPPTXPresentation(title, subtitle string, slides []pptxSlide, themeName string) plannedPPTXPresentation {
	if strings.TrimSpace(themeName) == "" {
		themeName = pickThemeName(title, subtitle)
	}

	planned := plannedPPTXPresentation{
		ThemeName: themeName,
		Slides:    make([]plannedPPTXSlide, 0, len(slides)),
	}
	for i, slide := range slides {
		planned.Slides = append(planned.Slides, planPPTXSlide(slide, title, i))
	}
	return planned
}

func planPPTXSlide(s pptxSlide, deckTitle string, index int) plannedPPTXSlide {
	layout := effectivePPTXLayout(s)
	planned := s
	planned.Type = layout

	if strings.TrimSpace(planned.Heading) == "" {
		planned.Heading = defaultSlideHeading(layout, deckTitle, index)
	}

	switch layout {
	case "stats":
		planned.Stats = effectiveStats(planned)
		if len(planned.Stats) == 0 {
			planned.Stats = []pptxStat{
				{Value: "92%", Label: "Retention", Desc: "Strong repeat usage"},
				{Value: "3.1x", Label: "ROI", Desc: "Efficiency uplift"},
				{Value: "14d", Label: "Payback", Desc: "Fast time to value"},
			}
		}
	case "steps":
		planned.Steps = effectiveSteps(planned)
		if len(planned.Steps) == 0 {
			planned.Steps = []string{"Clarify the goal", "Map the workflow", "Build the path", "Measure the result"}
		}
	case "cards":
		planned.Cards = effectiveCards(planned)
		for i := range planned.Cards {
			if strings.TrimSpace(planned.Cards[i].Icon) == "" {
				planned.Cards[i].Icon = inferCardIcon(planned.Cards[i], i)
			}
		}
	case "chart":
		planned.ChartSeries = effectiveChartSeries(planned)
		planned.ChartCategories = chartCategoriesOrFallback(planned)
		if strings.TrimSpace(planned.ChartType) == "" {
			planned.ChartType = "column"
		}
	case "timeline":
		if len(planned.Timeline) == 0 {
			planned.Timeline = []pptxTimelineItem{
				{Date: "Q1", Title: "Discovery", Desc: "Frame the opportunity"},
				{Date: "Q2", Title: "Build", Desc: "Launch the first version"},
				{Date: "Q3", Title: "Adopt", Desc: "Drive usage"},
				{Date: "Q4", Title: "Scale", Desc: "Expand the rollout"},
			}
		}
	case "compare":
		left, right := effectiveCompareColumns(planned)
		planned.LeftColumn = &left
		planned.RightColumn = &right
	case "table":
		if planned.Table == nil || len(planned.Table.Headers) == 0 {
			planned.Table = &pptxTableData{
				Headers: []string{"Option", "Time", "Cost"},
				Rows: [][]string{
					{"Manual", "5 days", "$80k"},
					{"Hybrid", "2 days", "$35k"},
					{"Automated", "4 hours", "$12k"},
				},
			}
		}
	case "bullets":
		planned.Points = safePoints(planned.Points, 6)
		if len(planned.Points) == 0 {
			planned.Points = []string{"Define the message", "Support it with evidence", "Close with an action"}
		}
	case "blank", "title":
		if len(planned.Points) == 0 {
			planned.Points = []string{"Transition to the next section"}
		}
	}

	return plannedPPTXSlide{
		Slide:  planned,
		Layout: layout,
	}
}

func defaultSlideHeading(layout, deckTitle string, index int) string {
	switch layout {
	case "title":
		return "Section"
	case "stats":
		return "Key Metrics"
	case "steps":
		return "How It Works"
	case "cards":
		return "Core Capabilities"
	case "chart":
		return "Trend Analysis"
	case "timeline":
		return "Roadmap"
	case "compare":
		return "Comparison"
	case "table":
		return "Decision Matrix"
	case "blank":
		return deckTitle
	default:
		return "Slide " + strconv.Itoa(index+2)
	}
}
