package types

import (
	"time"
)

// ChartDetails contains details of a chart
type ChartDetails struct {
	PublishedAt time.Time
	Digest      string
}
