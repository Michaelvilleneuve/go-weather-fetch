package forecast

import (
	"sync"

	"github.com/Michaelvilleneuve/weather-fetch-go/internal/arome"
)

func StartFetching() {
	var wg sync.WaitGroup
	
	for _, forecastPackage := range arome.Configuration().Packages {
		wg.Add(1)
		go func(fp arome.AromePackage) {
			defer wg.Done()
			fp.ProcessLatestRun()
		}(forecastPackage)
	}
	
	wg.Wait()

	StartFetching()
}





