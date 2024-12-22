package styles

import (
	"github.com/AccentDesign/gcss"
	"github.com/AccentDesign/gcss/props"
	"github.com/a-h/templ"
)

var (
	PostCardStyles = Stylesheet{
		{
			Selector: ".postcard aside:hover",
			Props:    gcss.Props{},
			CustomProps: []gcss.CustomProp{
				{
					Attr:  "box-shadow",
					Value: "0 2px 4px rgba(0,0,0,0.1)",
				},
			},
		},
		{
			Selector: ".postcard h2",
			Props: gcss.Props{
				FontSize: props.Unit{Size: 1.5, Type: props.UnitTypeEm},
				Margin:   props.Unit{Size: 0, Type: props.UnitTypeAuto},
			},
		},
		{
			Selector: ".postcard p",
			Props: gcss.Props{
				FontSize: props.Unit{Size: 1, Type: props.UnitTypeEm},
				Margin:   props.Unit{Size: 0, Type: props.UnitTypeAuto},
			},
		},
		{
			Selector: ".postcard a",
			Props: gcss.Props{
				TextDecorationLine: props.TextDecorationLineNone,
				Color:              props.ColorRGBA(51, 51, 51, 255),
			},
		},
	}
	PostCardStylesHandle = templ.NewOnceHandle()
)
