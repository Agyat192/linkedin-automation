package stealth

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/go-rod/rod"
	"github.com/sirupsen/logrus"
)

type MouseController struct {
	config MouseMovementConfig
	logger *logrus.Logger
	rng    *rand.Rand
}

func NewMouseController(config MouseMovementConfig, logger *logrus.Logger) *MouseController {
	return &MouseController{
		config: config,
		logger: logger,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (mc *MouseController) IntelligentClick(page *rod.Page, selector string) error {
	element, err := page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found: %w", err)
	}
	
	time.Sleep(time.Duration(mc.rng.Float64()*500+200) * time.Millisecond)
	
	if err := element.Click("left", 1); err != nil {
		return fmt.Errorf("failed to click element: %w", err)
	}
	
	time.Sleep(time.Duration(mc.rng.Float64()*200+100) * time.Millisecond)
	return nil
}

func (mc *MouseController) IntelligentScroll(page *rod.Page, direction string, amount int) error {
	chunkSize := 3
	scrollsNeeded := amount / chunkSize
	
	for i := 0; i < scrollsNeeded; i++ {
		time.Sleep(time.Duration(mc.rng.Float64()*200+100) * time.Millisecond)
		
		switch direction {
		case "down":
			page.Scroll(-float64(chunkSize), 0)
		case "up":
			page.Scroll(float64(chunkSize), 0)
		}
		
		time.Sleep(time.Duration(mc.rng.Float64()*100+50) * time.Millisecond)
	}
	return nil
}

func (mc *MouseController) IntelligentHover(page *rod.Page, selector string, duration time.Duration) error {
	element, err := page.Element(selector)
	if err != nil {
		return fmt.Errorf("element not found for hover: %w", err)
	}
	
	if err := element.Hover(); err != nil {
		return fmt.Errorf("failed to hover element: %w", err)
	}
	
	time.Sleep(duration)
	return nil
}
