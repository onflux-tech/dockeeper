package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const textFieldMax = 1_000_000

func init() {
	m.Register(func(app core.App) error {
		return bumpTextLimits(app)
	}, func(app core.App) error {
		return resetTextLimits(app)
	})
}

func bumpTextLimits(app core.App) error {
	return setTextLimits(app, 0, textFieldMax)
}

func resetTextLimits(app core.App) error {
	return setTextLimits(app, textFieldMax, 0)
}

func setTextLimits(app core.App, from, to int) error {
	collections, err := app.FindAllCollections()
	if err != nil {
		return err
	}

	for _, col := range collections {
		if col.System {
			continue
		}
		changed := false
		for _, field := range col.Fields {
			text, ok := field.(*core.TextField)
			if !ok || text.Max != from {
				continue
			}
			text.Max = to
			changed = true
		}
		if changed {
			if err := app.Save(col); err != nil {
				return err
			}
		}
	}

	return nil
}
