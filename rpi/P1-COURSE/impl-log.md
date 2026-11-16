# P1-COURSE impl log
- pdf.go and course.go. `go vet` clean.
- course_test.go: PDF pages and images, bad file, 201 pages, paste, empty page, edit then Ready then not Ready, quote spanning a line break, non-Ready page quote refused, un-ready refused while a claim anchors, other user gets not found. `go test -race ./internal/course` green.
