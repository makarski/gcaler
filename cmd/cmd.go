package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"google.golang.org/api/calendar/v3"

	"github.com/makarski/gcaler/config"
	gcal "github.com/makarski/gcaler/google/calendar"
	"github.com/makarski/gcaler/userio"
)

var Out = os.Stdout

type (
	CmdFunc func(gcal.GCalendar) error
)

func CalSrvLocation(
	ctx context.Context,
	gCalendar *gcal.GCalendar,
	template *config.Template,
) (*calendar.Service, *time.Location, error) {
	calSrv, err := gCalendar.CalendarService(ctx, handleAuthConsent)
	if err != nil {
		return nil, nil, err
	}

	tz, err := time.LoadLocation(template.Timezone)
	if err != nil {
		return nil, nil, err
	}

	return calSrv, tz, nil
}

func handleAuthConsent(authURL string) (string, error) {
	fmt.Fprintf(Out, "> Visit the link: %v\n", authURL)
	return userio.UserIn(bytes.NewBufferString("> Enter auth. code: "))
}
