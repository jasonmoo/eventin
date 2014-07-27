package main

import (
	"html/template"
	"strings"
	"time"
)

const EventHTML = `
<button>&times;</button>
<div class="event">
	<h1>{{.EventName}}</h1>

	{{if (or .Borough (or .Neighborhood (or .Category (or .Subcategory))))}}
	<p>{{concat " / " .Borough .Neighborhood .Category .Subcategory}}</p>
	{{end}}

	{{if .DateTimeDescription}}<p>When: {{.DateTimeDescription}}</p>{{end}}
	{{if .WebDescription}}<blockquote>{{puthtml .WebDescription}}</blockquote>{{end}}
	{{if .EventDetailUrl}}<p>Details: <a href="{{httpit .EventDetailUrl}}" target="_blank">{{.EventDetailUrl}}</a></p>{{end}}

	{{if .VenueName}}<h2>Venue: {{.VenueName}}</h2>{{end}}
	{{if .VenueWebsite}}<p>Website: <a href="{{httpit .VenueWebsite}}" target="_blank">{{.VenueWebsite}}</a></p>{{end}}
	{{if .VenueDetailUrl}}<p>Details: <a href="{{httpit .VenueDetailUrl}}" target="_blank">{{.VenueDetailUrl}}</a></p>{{end}}
	{{if .StreetAddress}}<p>Address: <a href="{{.MapURL}}" target="_blank">{{.StreetAddress}}</a></p>{{end}}
	{{if .Telephone}}<p>Telephone: <a href="tel:{{.Telephone}}">{{.Telephone}}</a></p>{{end}}


	{{if (or .Festival (or .Free (or .KidFriendly (or .LastChance (or .LongRunningShow (or .TimesPick (or .PreviewsAndOpenings)))))))}}
	<p>
		Details:
		{{if .Festival}}<span>Festival<span>{{end}}
		{{if .Free}}<span>Free<span>{{end}}
		{{if .KidFriendly}}<span>KidFriendly<span>{{end}}
		{{if .LastChance}}<span>LastChance<span>{{end}}
		{{if .LongRunningShow}}<span>LongRunningShow<span>{{end}}
		{{if .TimesPick}}<span>TimesPick<span>{{end}}
		{{if .PreviewsAndOpenings}}<span>PreviewsAndOpenings<span>{{end}}
	</p>
	{{end}}

	{{if .RecurDays}}
		<p>Days: {{range .RecurDays}}<span>{{.}}</span>{{end}} (since {{shortdate .RecurringStartDate}})</p>
	{{end}}

	<p class="subdetails">Last Modified: {{nicedate .LastModified}}</p>
</div>
`

var EventTemplate *template.Template

func init() {
	EventTemplate = template.New("Event template")
	EventTemplate.Funcs(template.FuncMap{
		"puthtml": func(s string) template.HTML { return template.HTML(s) },
		"concat": func(sep string, args ...string) string {
			a := make([]string, 0)
			for _, arg := range args {
				if len(arg) > 0 {
					a = append(a, arg)
				}
			}
			return strings.Join(a, sep)
		},
		"nicedate": func(ts time.Time) string {
			return ts.Format("January _2 3:04:05 PM 2006")
		},
		"shortdate": func(ts time.Time) string {
			return ts.Format("January _2 2006")
		},
		"httpit": func(url string) string {
			if !strings.HasPrefix(url, "http") {
				return "http://" + url
			}
			return url
		},
	})
	template.Must(EventTemplate.Parse(EventHTML))
}
