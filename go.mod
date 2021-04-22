module github.com/nm-morais/x-bot

go 1.15

require (
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20210208195552-ff826a37aa15 // indirect
	github.com/nm-morais/go-babel v1.0.1
	github.com/sirupsen/logrus v1.8.1
	github.com/ungerik/go-dry v0.0.0-20210209114055-a3e162a9e62e
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/nm-morais/go-babel => ../go-babel
