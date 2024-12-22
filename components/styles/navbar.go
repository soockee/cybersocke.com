package styles

import (
	"github.com/AccentDesign/gcss"
	"github.com/AccentDesign/gcss/props"
	"github.com/a-h/templ"
)

var (
	NavbarStyles = Stylesheet{
		{
			Selector: ".navbar",
			Props: gcss.Props{
				Display:         props.DisplayFlex,
				JustifyContent:  props.JustifyContentCenter,
				BackgroundColor: primaryColor,
				AlignItems:      props.AlignItemsCenter,
				Height: props.Unit{
					Size: 100,
					Type: props.UnitTypePx,
				},
			},
			CustomProps: []gcss.CustomProp{
				{
					Attr:  "box-shadow",
					Value: "0 2px 4px rgba(0,0,0,0.1)",
				},
			},
		},
		{
			Selector: ".navbar ul",
			Props: gcss.Props{
				ListStyleType: props.ListStyleTypeNone,
				Padding:       props.Unit{Size: 0, Type: props.UnitTypeAuto},
				Margin:        props.Unit{Size: 0, Type: props.UnitTypeAuto},
				Display:       props.DisplayFlex,
				Gap:           props.Unit{Size: 30, Type: props.UnitTypePx},
			},
		},
		{
			Selector: ".navbar li",
			Props: gcss.Props{
				Display: props.DisplayInline,
			},
		},
		{
			Selector: ".navbar a",
			Props: gcss.Props{
				Color:              secondaryColor,
				FontSize:           props.Unit{Size: 1.2, Type: props.UnitTypeEm},
				FontWeight:         props.FontWeightBold,
				TextDecorationLine: props.TextDecorationLineNone,
				BorderRadius:       props.Unit{Size: 5, Type: props.UnitTypePx},
				PaddingBottom:      props.Unit{Size: 10, Type: props.UnitTypePx},
				PaddingTop:         props.Unit{Size: 10, Type: props.UnitTypePx},
				PaddingLeft:        props.Unit{Size: 20, Type: props.UnitTypePx},
				PaddingRight:       props.Unit{Size: 20, Type: props.UnitTypePx},
				Position:           props.PositionRelative,
				Overflow:           props.OverflowHidden,
			},
			CustomProps: []gcss.CustomProp{
				{
					Attr:  "transition",
					Value: "all 0.3s ease",
				},
			},
		},
		{
			Selector: ".navbar a:before",
			Props: gcss.Props{
				BackgroundColor: props.ColorRGBA(255, 255, 255, 20),
				Position:        props.PositionAbsolute,
				ZIndex:          props.Unit{Size: 0, Type: props.UnitTypeRaw},
				Top:             props.Unit{Size: 0, Type: props.UnitTypeRaw},
				Left:            props.Unit{Size: 0, Type: props.UnitTypeRaw},
				Width:           props.Unit{Size: 100, Type: props.UnitTypePx},
				Height:          props.Unit{Size: 100, Type: props.UnitTypePx},
			},
			CustomProps: []gcss.CustomProp{
				{
					Attr:  "content",
					Value: "\"\"",
				},
				{
					Attr:  "transform",
					Value: "scaleX(0)",
				},
				{
					Attr:  "transform-origin",
					Value: "right",
				},
				{
					Attr:  "transition",
					Value: "all 0.3s ease",
				},
			},
		},
		{
			Selector: ".navbar a:hover:before",
			CustomProps: []gcss.CustomProp{
				{
					Attr:  "transform",
					Value: "scaleX(1)",
				},
				{
					Attr:  "transform-origin",
					Value: "left",
				},
			},
		},
		{
			Selector: ".navbar a:hover",
			Props: gcss.Props{
				Color:           primaryColor,
				BackgroundColor: secondaryColor,
				Cursor:          props.CursorPointer,
			},

			CustomProps: []gcss.CustomProp{
				{
					Attr:  "box-shadow",
					Value: "0 4px 8px rgba(0, 0, 0, 0.4)",
				},
			},
		},
	}
	NavbarStylesHandle = templ.NewOnceHandle()
)
