module news-pulse-backend

go 1.24.0

toolchain go1.24.7

require (
	github.com/sirupsen/logrus v1.9.3
	news-parsing-service v0.0.0
)

replace news-parsing-service => ./news-parsing-service

require (
	github.com/PuerkitoBio/goquery v1.8.0 // indirect
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/andybalholm/cascadia v1.3.3 // indirect
	github.com/araddon/dateparse v0.0.0-20210429162001-6b43995a97de // indirect
	github.com/go-shiori/dom v0.0.0-20230515143342-73569d674e1c // indirect
	github.com/go-shiori/go-readability v0.0.0-20250217085726-9f5bf5ca7612 // indirect
	github.com/gogs/chardet v0.0.0-20211120154057-b7413eaefb8f // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kljensen/snowball v0.10.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/mmcdole/gofeed v1.3.0 // indirect
	github.com/mmcdole/goxpp v1.1.1-0.20240225020742-a0c311522b23 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	golang.org/x/net v0.44.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
