package spinner

import (
	"time"

	"github.com/briandowns/spinner"
)

func WithLoadingSpinner(style int, delay time.Duration, action func()) {
	var (
		s    = spinner.New(spinner.CharSets[style], delay)
		done = make(chan struct{})
	)

	s.Prefix = "\n"
	s.Suffix = "   Loading"

	s.Start()

	defer func(chan struct{}) {
		s.Stop()
		close(done)
	}(done)

	go func() {
		dots := []string{"", ".", "..", "..."}
		i := 0

		for {

			select {

			case <-done:
				return

			case <-time.After(300 * time.Millisecond):
				s.Suffix = "   Loading" + dots[i%len(dots)]
				i++
			}
		}
	}()

	action()
}
