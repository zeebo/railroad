package railroad

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
)

var replacer = strings.NewReplacer(
	`*`, `&#42;`,
	`_`, `&#95;`,
	"`", `&#96;`,
	`[`, `&#91;`,
	`]`, `&#93;`,
	`<`, `&#60;`,
	`&`, `&#38;`,
)

func e(x string) string { return replacer.Replace(x) }

// Text embeds the string as a RailItem.
func Text(text string) RailItem { return Terminal(text) }

// global config. deal with it
var (
	ConfigVerticalSeparation = 8.0
	ConfigArcRadius          = 10.0
	ConfigDiagramClass       = "railroad-diagram"
	ConfigTranslateHalfPixel = true
	ConfigInternalAlignment  = "center"
	ConfigDebug              = false
	ConfigCharacterAdvance   = 8.0
	ConfigDefaultStyle       = `    svg.railroad-diagram {
        background-color:hsl(30,20%,95%);
    }
    svg.railroad-diagram path {
        stroke-width:3;
        stroke:black;
        fill:rgba(0,0,0,0);
    }
    svg.railroad-diagram text {
        font:bold 14px monospace;
        text-anchor:middle;
    }
    svg.railroad-diagram text.label{
        text-anchor:start;
    }
    svg.railroad-diagram text.comment{
        font:italic 12px monospace;
    }
    svg.railroad-diagram rect{
        stroke-width:3;
        stroke:black;
        fill:hsl(120,100%,90%);
    }
`
)

type textItem string

func (t textItem) writeSvg(write func(string, ...interface{})) { write("%s", e(string(t))) }
func (t textItem) format(x, y, width float64) RailItem         { panic("virtual") }
func (t textItem) getWidth() float64                           { panic("virtual") }
func (t textItem) getHeight() float64                          { panic("virtual") }
func (t textItem) getUp() float64                              { panic("virtual") }
func (t textItem) getDown() float64                            { panic("virtual") }
func (t textItem) getNeedsSpace() bool                         { panic("virtual") }
func (t textItem) addChild(ch RailItem)                        { panic("virtual") }

func max(x, y float64) float64 { return math.Max(x, y) }

func determineGaps(outer, inner float64) (float64, float64) {
	diff := outer - inner
	if ConfigInternalAlignment == "left" {
		return 0, diff
	} else if ConfigInternalAlignment == "right" {
		return diff, 0
	} else {
		return float64(int(diff) / 2), float64(int(diff) / 2)
	}
}

type RailItem interface {
	format(x, y, width float64) RailItem
	writeSvg(write func(string, ...interface{}))
	getWidth() float64
	getHeight() float64
	getUp() float64
	getDown() float64
	getNeedsSpace() bool
	addChild(ch RailItem)
}

type diagramItem struct {
	name          string
	width, height float64
	up, down      float64
	attrs         a
	children      []RailItem
	needsSpace    bool
}

type a map[string]string

func newDiagramText(name string, text string, attrs a) *diagramItem {
	if attrs == nil {
		attrs = make(a)
	}
	return &diagramItem{
		name:     name,
		attrs:    attrs,
		children: []RailItem{textItem(text)},
	}
}

func newDiagramItem(name string, attrs a) *diagramItem {
	if attrs == nil {
		attrs = make(a)
	}
	return &diagramItem{
		name:  name,
		attrs: attrs,
	}
}

func (self *diagramItem) getWidth() float64    { return self.width }
func (self *diagramItem) getHeight() float64   { return self.height }
func (self *diagramItem) getUp() float64       { return self.up }
func (self *diagramItem) getDown() float64     { return self.down }
func (self *diagramItem) getNeedsSpace() bool  { return self.needsSpace }
func (self *diagramItem) addChild(ch RailItem) { self.children = append(self.children, ch) }

func (self *diagramItem) format(x, y, width float64) RailItem { panic("virtual") }

func (self *diagramItem) writeSvg(write func(string, ...interface{})) {
	write(`<%s`, self.name)
	keys := make([]string, 0, len(self.attrs))
	for key := range self.attrs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		write(` %s="%s"`, key, e(self.attrs[key]))
	}
	write(`>`)
	if self.name == "g" || self.name == "svg" {
		write("\n")
	}
	for _, child := range self.children {
		child.writeSvg(write)
	}
	write(`</%s>`, self.name)
}

type path struct {
	*diagramItem
	x, y float64
}

func newPath(x, y float64) *path {
	return &path{
		diagramItem: newDiagramItem("path", a{
			"d": fmt.Sprintf("M%v %v", x, y),
		}),
		x: x,
		y: y,
	}
}

func (self *path) m(x, y float64) *path {
	self.attrs["d"] += fmt.Sprintf("m%v %v", x, y)
	return self
}

func (self *path) h(val float64) *path {
	if val == 0 {
		val = 0
	}
	self.attrs["d"] += fmt.Sprintf("h%v", val)
	return self
}

func (self *path) right(val float64) *path {
	return self.h(max(0, val))
}

func (self *path) left(val float64) *path {
	return self.h(-max(0, val))
}

func (self *path) v(val float64) *path {
	if val == 0 {
		val = 0
	}
	self.attrs["d"] += fmt.Sprintf("v%v", val)
	return self
}

func (self *path) down(val float64) *path {
	return self.v(max(0, val))
}

func (self *path) up(val float64) *path {
	return self.v(-max(0, val))
}

func (self *path) arc(sweep string) *path {
	x := ConfigArcRadius
	y := ConfigArcRadius
	if sweep[0] == 'e' || sweep[1] == 'w' {
		x *= -1
	}
	if sweep[0] == 's' || sweep[1] == 'n' {
		y *= -1
	}
	cw := 0
	if sweep == "ne" || sweep == "es" || sweep == "sw" || sweep == "wn" {
		cw = 1
	}
	self.attrs["d"] += fmt.Sprintf(`a%[1]v %[1]v 0 0 %[2]v %[3]v %[4]v`, ConfigArcRadius, cw, x, y)
	return self
}

func (self *path) format(x, y, width float64) RailItem {
	self.attrs["d"] += "h.5"
	return self
}

type style struct {
	*diagramItem
	css string
}

func Style(css string) RailItem {
	di := newDiagramItem("style", nil)
	return &style{
		diagramItem: di,
		css:         css,
	}
}

func (self *style) writeSvg(write func(string, ...interface{})) {
	cdata := fmt.Sprintf("/* <![CDATA[ */\n%s\n/* ]]> */\n", self.css)
	write(`<style>%s</style>`, cdata)
}

func (self *style) getWidth() float64                   { return 0 }
func (self *style) getHeight() float64                  { return 0 }
func (self *style) getUp() float64                      { return self.up }
func (self *style) getDown() float64                    { return self.down }
func (self *style) getNeedsSpace() bool                 { return false }
func (self *style) format(x, y, width float64) RailItem { return self }

type diagram struct {
	*diagramItem
	type_     string
	css       string
	items     []RailItem
	formatted bool
}

func Diagram(items ...RailItem) io.WriterTo {
	di := newDiagramItem("svg", a{
		"class": ConfigDiagramClass,
	})
	// TODO kwargs
	css := ConfigDefaultStyle
	var items_ []RailItem
	if css != "" {
		items_ = append(items_, Style(css))
	}
	items_ = append(items_, newStart())
	items_ = append(items_, items...)
	items_ = append(items_, newEnd())

	for _, item := range items_ {
		if _, ok := item.(*style); ok {
			continue
		}
		di.width += item.getWidth()
		if item.getNeedsSpace() {
			di.width += 20
		}
		di.up = max(di.up, item.getUp()-di.height)
		di.height += item.getHeight()
		di.down = max(di.down-item.getHeight(), item.getDown())
	}
	if items_[0].getNeedsSpace() {
		di.width -= 10
	}
	if items_[len(items_)-1].getNeedsSpace() {
		di.width -= 10
	}

	return &diagram{
		diagramItem: di,
		css:         css,
		items:       items_,
		formatted:   false,
	}
}

func (d *diagram) WriteTo(w io.Writer) (n int64, err error) {
	d.writeSvg(func(format string, args ...interface{}) {
		var ni int
		ni, err = fmt.Fprintf(w, format, args...)
		n += int64(ni)
	})
	return n, err
}

// TODO padding
func (self *diagram) format(x, y, width float64) RailItem {
	paddingTop := 20.0
	paddingRight := paddingTop
	paddingBottom := paddingTop
	paddingLeft := paddingRight

	x = paddingLeft
	y = paddingTop + self.up
	g := newDiagramItem("g", nil)
	if ConfigTranslateHalfPixel {
		g.attrs["transform"] = "translate(.5 .5)"
	}
	for _, item := range self.items {
		if item.getNeedsSpace() {
			g.addChild(newPath(x, y).h(10))
			x += 10
		}
		g.addChild(item.format(x, y, item.getWidth()))
		x += item.getWidth()
		y += item.getHeight()
		if item.getNeedsSpace() {
			g.addChild(newPath(x, y).h(10))
			x += 10
		}
	}
	self.attrs["width"] = fmt.Sprint(self.width + paddingLeft + paddingRight)
	self.attrs["height"] = fmt.Sprint(self.up + self.height + self.down + paddingTop + paddingBottom)
	self.attrs["viewBox"] = fmt.Sprintf("0 0 %s %s", self.attrs["width"], self.attrs["height"])
	self.addChild(g)
	self.formatted = true
	return self
}

func (self *diagram) writeSvg(write func(string, ...interface{})) {
	if !self.formatted {
		self.format(0, 0, 0)
	}
	self.diagramItem.writeSvg(write)
}

type sequence struct {
	*diagramItem
	items []RailItem
}

func Sequence(items ...RailItem) RailItem {
	di := newDiagramItem("g", nil)
	di.needsSpace = true
	di.up = 0
	di.down = 0
	di.height = 0
	di.width = 0
	for _, item := range items {
		di.width += item.getWidth()
		if item.getNeedsSpace() {
			di.width += 20
		}
		di.up = max(di.up, item.getUp()-di.height)
		di.height += item.getHeight()
		di.down = max(di.down-item.getHeight(), item.getDown())
	}
	if items[0].getNeedsSpace() {
		di.width -= 10
	}
	if items[len(items)-1].getNeedsSpace() {
		di.width -= 10
	}
	// TODO debug
	return &sequence{
		diagramItem: di,
		items:       items,
	}
}

func (self *sequence) format(x, y, width float64) RailItem {
	leftGap, rightGap := determineGaps(width, self.width)
	self.addChild(newPath(x, y).h(leftGap))
	self.addChild(newPath(x+leftGap+self.width, y+self.height).h(rightGap))
	x += leftGap
	for i, item := range self.items {
		if item.getNeedsSpace() && i > 0 {
			self.addChild(newPath(x, y).h(10))
			x += 10
		}
		self.addChild(item.format(x, y, item.getWidth()))
		x += item.getWidth()
		y += item.getHeight()
		if item.getNeedsSpace() && i < len(self.items)-1 {
			self.addChild(newPath(x, y).h(10))
			x += 10
		}
	}
	return self
}

type stack struct {
	*diagramItem
	items []RailItem
}

func Stack(items ...RailItem) RailItem {
	di := newDiagramItem("g", nil)
	di.needsSpace = true
	for _, item := range items {
		w := item.getWidth()
		if item.getNeedsSpace() {
			w += 20
		}
		di.width = max(di.width, w)
	}
	if len(items) > 1 { // python code is pretty sure this calc is totes wrong
		di.width += ConfigArcRadius * 2
	}
	di.up = items[0].getUp()
	di.down = items[len(items)-1].getDown()
	di.height = 0
	last := len(items) - 1
	for i, item := range items {
		di.height += item.getHeight()
		if i > 0 {
			di.height += max(ConfigArcRadius*2, item.getUp()+ConfigVerticalSeparation)
		}
		if i < last {
			di.height += max(ConfigArcRadius*2, item.getDown()+ConfigVerticalSeparation)
		}
	}
	// TODO debug
	return &stack{
		diagramItem: di,
		items:       items,
	}
}

func (self *stack) format(x, y, width float64) RailItem {
	leftGap, rightGap := determineGaps(width, self.width)
	self.addChild(newPath(x, y).h(leftGap))
	x += leftGap
	xInitial := x
	innerWidth := 0.0
	if len(self.items) > 1 {
		self.addChild(newPath(x, y).h(ConfigArcRadius))
		x += ConfigArcRadius
		innerWidth = self.width - ConfigArcRadius*2
	} else {
		innerWidth = self.width
	}
	for i, item := range self.items {
		self.addChild(item.format(x, y, innerWidth))
		x += innerWidth
		y += item.getHeight()
		if i != len(self.items)-1 {
			self.addChild(newPath(x, y).
				arc("ne").down(max(0, item.getDown()+ConfigVerticalSeparation-2*ConfigArcRadius)).
				arc("es").left(innerWidth).
				arc("nw").down(max(0, self.items[i+1].getUp()+ConfigVerticalSeparation-ConfigArcRadius*2)).
				arc("ws"))
			y += max(item.getDown()+ConfigVerticalSeparation, ConfigArcRadius*2) +
				max(self.items[i+1].getUp()+ConfigVerticalSeparation, ConfigArcRadius*2)
			x = xInitial + ConfigArcRadius
		}
	}
	if len(self.items) > 1 {
		self.addChild(newPath(x, y).h(ConfigArcRadius))
		x += ConfigArcRadius
	}
	self.addChild(newPath(x, y).h(rightGap))
	return self
}

type optionalSequence struct {
	*diagramItem
	items []RailItem
}

func OptionalSequence(items ...RailItem) RailItem {
	if len(items) <= 1 {
		return Sequence(items...)
	}

	di := newDiagramItem("g", nil)
	di.needsSpace = false
	di.width = 0
	di.up = 0
	di.height = 0
	for _, item := range items {
		di.height += item.getHeight()
	}
	di.down = items[0].getDown()
	heightSoFar := 0.0
	for i, item := range items {
		di.up = max(di.up, max(ConfigArcRadius*2, item.getUp()+ConfigVerticalSeparation)-heightSoFar)
		heightSoFar += item.getHeight()
		if i > 0 {
			di.down = max(di.height+di.down, heightSoFar+
				max(ConfigArcRadius*2, item.getDown()+ConfigVerticalSeparation)) - di.height
		}
		itemWidth := item.getWidth()
		if item.getNeedsSpace() {
			itemWidth += 20
		}
		if i == 0 {
			di.width += ConfigArcRadius + max(itemWidth, ConfigArcRadius)
		} else {
			di.width += ConfigArcRadius*2 + max(itemWidth, ConfigArcRadius) + ConfigArcRadius
		}
	}
	// TODO debug
	return &optionalSequence{
		diagramItem: di,
		items:       items,
	}
}

func (self *optionalSequence) format(x, y, width float64) RailItem {
	leftGap, rightGap := determineGaps(width, self.width)
	self.addChild(newPath(x, y).right(leftGap))
	self.addChild(newPath(x+leftGap+self.width, y+self.height).right(rightGap))
	x += leftGap
	upperLineY := y - self.up
	last := len(self.items) - 1
	for i, item := range self.items {
		itemSpace := 0.0
		if item.getNeedsSpace() {
			itemSpace = 10
		}
		itemWidth := item.getWidth() + itemSpace

		if i == 0 {
			self.addChild(newPath(x, y).
				arc("se").up(y - upperLineY - ConfigArcRadius*2).
				arc("wn").right(itemWidth - ConfigArcRadius).
				arc("ne").down(y + item.getHeight() - upperLineY - ConfigArcRadius*2).
				arc("ws"))
			self.addChild(newPath(x, y).right(itemSpace + ConfigArcRadius))
			self.addChild(item.format(x+itemSpace+ConfigArcRadius, y, item.getWidth()))
			x += itemWidth + ConfigArcRadius
			y += item.getHeight()
		} else if i < last {
			self.addChild(newPath(x, upperLineY).
				right(ConfigArcRadius*2 + max(itemWidth, ConfigArcRadius) + ConfigArcRadius).
				arc("ne").
				down(y - upperLineY + item.getHeight() - ConfigArcRadius*2).
				arc("ws"))
			self.addChild(newPath(x, y).right(ConfigArcRadius * 2))
			self.addChild(item.format(x+ConfigArcRadius*2, y, item.getWidth()))
			self.addChild(newPath(x+item.getWidth()+ConfigArcRadius*2, y+item.getHeight()).
				right(itemSpace + ConfigArcRadius))
			self.addChild(newPath(x, y).
				arc("ne").down(item.getHeight() + max(item.getDown()+ConfigVerticalSeparation, ConfigArcRadius*2) - ConfigArcRadius*2).
				arc("ws").right(itemWidth - ConfigArcRadius).
				arc("se").up(item.getDown() + ConfigVerticalSeparation - ConfigArcRadius*2).
				arc("wn"))
			x += ConfigArcRadius*2 + max(itemWidth, ConfigArcRadius) + ConfigArcRadius
			y += item.getHeight()
		} else {
			self.addChild(newPath(x, y).right(ConfigArcRadius * 2))
			self.addChild(item.format(x+ConfigArcRadius*2, y, item.getWidth()))
			self.addChild(newPath(x+ConfigArcRadius*2+item.getWidth(), y+item.getHeight()).
				right(itemSpace + ConfigArcRadius))
			self.addChild(newPath(x, y).
				arc("ne").down(item.getHeight() + max(item.getDown()+ConfigVerticalSeparation, ConfigArcRadius*2) - ConfigArcRadius*2).
				arc("ws").right(itemWidth - ConfigArcRadius).
				arc("se").up(item.getDown() + ConfigVerticalSeparation - ConfigArcRadius*2).
				arc("wn"))
		}
	}
	return self
}

type choice struct {
	*diagramItem
	def   int
	items []RailItem
}

func Choice(default_ int, items ...RailItem) RailItem {
	di := newDiagramItem("g", nil)
	for _, item := range items {
		di.width = max(di.width, item.getWidth())
	}
	di.width += ConfigArcRadius * 4
	di.up = items[0].getUp()
	di.down = items[len(items)-1].getDown()
	di.height = items[default_].getHeight()
	for i, item := range items {
		arcs := ConfigArcRadius
		if i == default_-1 || i == default_+1 {
			arcs = ConfigArcRadius * 2
		}
		if i < default_ {
			di.up += max(arcs, item.getHeight()+item.getDown()+
				ConfigVerticalSeparation+items[i+1].getUp())
		} else if i == default_ {
			continue
		} else {
			di.down += max(arcs, item.getUp()+ConfigVerticalSeparation+
				items[i-1].getDown()+items[i-1].getHeight())
		}
	}
	di.down -= items[default_].getHeight()
	return &choice{
		diagramItem: di,
		def:         default_,
		items:       items,
	}
}

func (self *choice) format(x, y, width float64) RailItem {
	leftGap, rightGap := determineGaps(width, self.width)

	self.addChild(newPath(x, y).h(leftGap))
	self.addChild(newPath(x+leftGap+self.width, y+self.height).h(rightGap))
	x += leftGap

	innerWidth := self.width - ConfigArcRadius*4
	def := self.items[self.def]

	above := self.items[:self.def]
	for i, j := 0, len(above)-1; i < j; i, j = i+1, j-1 {
		above[i], above[j] = above[j], above[i]
	}
	var distanceFromY float64
	if len(above) > 0 {
		distanceFromY = max(
			ConfigArcRadius*2,
			def.getUp()+
				ConfigVerticalSeparation+
				above[0].getDown()+
				above[0].getHeight())
	}
	for i, item := range above {
		ni := i - len(above)
		self.addChild(newPath(x, y).arc("se").up(distanceFromY - ConfigArcRadius*2).arc("wn"))
		self.addChild(item.format(x+ConfigArcRadius*2, y-distanceFromY, innerWidth))
		self.addChild(newPath(x+ConfigArcRadius*2+innerWidth, y-distanceFromY+item.getHeight()).
			arc("ne").down(distanceFromY - item.getHeight() + def.getHeight() - ConfigArcRadius*2).
			arc("ws"))
		if ni < -1 {
			distanceFromY += max(
				ConfigArcRadius,
				item.getUp()+
					ConfigVerticalSeparation+
					above[i+1].getDown()+
					above[i+1].getHeight())
		}
	}

	self.addChild(newPath(x, y).right(ConfigArcRadius * 2))
	self.addChild(self.items[self.def].format(x+ConfigArcRadius*2, y, innerWidth))
	self.addChild(newPath(x+ConfigArcRadius*2+innerWidth, y+self.height).
		right(ConfigArcRadius * 2))

	below := self.items[self.def+1:]
	if len(below) > 0 {
		distanceFromY = max(
			ConfigArcRadius*2,
			def.getHeight()+
				def.getDown()+
				ConfigVerticalSeparation+
				below[0].getUp())
	}
	for i, item := range below {
		self.addChild(newPath(x, y).arc("ne").down(distanceFromY - ConfigArcRadius*2).arc("ws"))
		self.addChild(item.format(x+ConfigArcRadius*2, y+distanceFromY, innerWidth))
		self.addChild(newPath(x+ConfigArcRadius*2+innerWidth, y+distanceFromY+item.getHeight()).
			arc("se").up(distanceFromY - ConfigArcRadius*2 + item.getHeight() - def.getHeight()).
			arc("wn"))
		belowAmount := 0.0
		if i+1 < len(below) {
			belowAmount = below[i+1].getUp()
		}
		distanceFromY += max(
			ConfigArcRadius,
			item.getHeight()+
				item.getDown()+
				ConfigVerticalSeparation+
				belowAmount)
	}
	return self
}

type multipleChoice struct {
	*diagramItem
	def        int
	type_      string
	items      []RailItem
	innerWidth float64
}

type MultipleChoiceType string

const (
	MultipleChoiceAny MultipleChoiceType = "any"
	MultipleChoiceAll MultipleChoiceType = "all"
)

func MultipleChoice(default_ int, type_ MultipleChoiceType, items ...RailItem) RailItem {
	di := newDiagramItem("g", nil)
	di.needsSpace = true
	innerWidth := 0.0
	for _, item := range items {
		innerWidth = max(innerWidth, item.getWidth())
	}
	di.width = 30 + ConfigArcRadius + innerWidth + ConfigArcRadius + 20
	di.up = items[0].getUp()
	di.down = items[len(items)-1].getDown()
	di.height = items[default_].getHeight()
	for i, item := range items {
		minimum := ConfigArcRadius
		if i == default_-1 || i == default_+1 {
			minimum = 10 + ConfigArcRadius
		}
		if i < default_ {
			di.up += max(minimum, item.getHeight()+item.getDown()+
				ConfigVerticalSeparation+items[i+1].getUp())
		} else if i == default_ {
			continue
		} else {
			di.down += max(minimum, item.getUp()+ConfigVerticalSeparation+
				items[i-1].getDown()+items[i-1].getHeight())
		}
	}
	di.down -= items[default_].getHeight()
	return &multipleChoice{
		diagramItem: di,
		def:         default_,
		type_:       string(type_),
		items:       items,
		innerWidth:  innerWidth,
	}
}

func (self *multipleChoice) format(x, y, width float64) RailItem {
	leftGap, rightGap := determineGaps(width, self.width)

	self.addChild(newPath(x, y).h(leftGap))
	self.addChild(newPath(x+leftGap+self.width, y+self.height).h(rightGap))
	x += leftGap

	def := self.items[self.def]

	above := self.items[:self.def]
	for i, j := 0, len(above)-1; i < j; i, j = i+1, j-1 {
		above[i], above[j] = above[j], above[i]
	}
	distanceFromY := 0.0
	if len(above) > 0 {
		distanceFromY = max(
			10+ConfigArcRadius,
			def.getUp()+
				ConfigVerticalSeparation+
				above[0].getDown()+
				above[0].getHeight())
	}
	for i, item := range above {
		ni := len(above) - i
		self.addChild(newPath(x+30, y).
			up(distanceFromY - ConfigArcRadius).
			arc("wn"))
		self.addChild(item.format(x+30+ConfigArcRadius, y-distanceFromY, self.innerWidth))
		self.addChild(newPath(x+30+ConfigArcRadius+self.innerWidth, y-distanceFromY+item.getHeight()).
			arc("ne").down(distanceFromY - item.getHeight() + def.getHeight() - ConfigArcRadius - 10))
		if ni < -1 {
			distanceFromY += max(
				ConfigArcRadius,
				item.getUp()+
					ConfigVerticalSeparation+
					above[i+1].getDown()+
					above[i+1].getHeight())
		}
	}

	self.addChild(newPath(x+30, y).right(ConfigArcRadius))
	self.addChild(self.items[self.def].format(x+30+ConfigArcRadius, y, self.innerWidth))
	self.addChild(newPath(x+30+ConfigArcRadius+self.innerWidth, y+self.height).right(ConfigArcRadius))

	below := self.items[self.def+1:]
	if len(below) > 0 {
		distanceFromY = max(
			10+ConfigArcRadius,
			def.getHeight()+
				def.getDown()+
				ConfigVerticalSeparation+
				below[0].getUp())
	}
	for i, item := range below {
		self.addChild(newPath(x+30, y).down(distanceFromY - ConfigArcRadius).arc("ws"))
		self.addChild(item.format(x+30+ConfigArcRadius, y+distanceFromY, self.innerWidth))
		self.addChild(newPath(x+30+ConfigArcRadius+self.innerWidth, y+distanceFromY+item.getHeight()).
			arc("se").up(distanceFromY - ConfigArcRadius + item.getHeight() - def.getHeight() - 10))
		belowAmount := 0.0
		if i+1 < len(below) {
			belowAmount = below[i+1].getUp()
		}
		distanceFromY += max(
			ConfigArcRadius,
			item.getHeight()+
				item.getDown()+
				ConfigVerticalSeparation+
				belowAmount)
	}

	branches := "take one or more branches, once each, in any order"
	amount := "1+"
	if self.type_ != "any" {
		branches = "take all branches, once each, in any order"
		amount = "all"
	}

	text := newDiagramItem("g", a{"class": "diagram-text"})
	text.addChild(newDiagramText("title", branches, nil))
	text.addChild(newDiagramItem("path", a{
		"d":     fmt.Sprintf("M %v %v h -26 a 4 4 0 0 0 -4 4 v 12 a 4 4 0 0 0 4 4 h 26 z", x+30, y-10),
		"class": "diagram-text",
	}))
	text.addChild(newDiagramText("text", amount, a{
		"x":     fmt.Sprint(x + 15),
		"y":     fmt.Sprint(y + 4),
		"class": "diagram-text",
	}))
	text.addChild(newDiagramItem("path", a{
		"d":     fmt.Sprintf("M %v %v h 16 a 4 4 0 0 1 4 4 v 12 a 4 4 0 0 1 -4 4 h -16 z", x+self.width-20, y-10),
		"class": "diagram-text",
	}))
	text.addChild(newDiagramText("text", "↺", a{
		"x":     fmt.Sprint(x + self.width - 10),
		"y":     fmt.Sprint(y + 4),
		"class": "diagram-arrow",
	}))
	self.addChild(text)
	return self
}

type OptionalOption struct {
	skip *bool
}

func OptionalSkip(skip bool) OptionalOption { return OptionalOption{skip: &skip} }

func Optional(item RailItem, options ...OptionalOption) RailItem {
	var skip bool
	for _, opt := range options {
		if opt.skip != nil {
			skip = *opt.skip
		}
	}

	which := 1
	if skip {
		which = 0
	}
	return Choice(which, Skip(), item)
}

type oneOrMore struct {
	*diagramItem
	item RailItem
	rep  RailItem
}

type OneOrMoreOption struct {
	repeat *RailItem
}

func OneOrMoreRepeat(repeat RailItem) OneOrMoreOption { return OneOrMoreOption{repeat: &repeat} }

func OneOrMore(item RailItem, options ...OneOrMoreOption) RailItem {
	var repeat RailItem
	for _, opt := range options {
		if opt.repeat != nil {
			repeat = *opt.repeat
		}
	}

	di := newDiagramItem("g", nil)
	if repeat == nil {
		repeat = Skip()
	}

	di.width = max(item.getWidth(), repeat.getWidth()) + ConfigArcRadius*2
	di.height = item.getHeight()
	di.up = item.getUp()
	di.down = max(ConfigArcRadius*2, item.getDown()+ConfigVerticalSeparation+repeat.getUp()+repeat.getHeight()+repeat.getDown())
	di.needsSpace = true
	// TODO debug
	return &oneOrMore{
		diagramItem: di,
		item:        item,
		rep:         repeat,
	}
}

func (self *oneOrMore) format(x, y, width float64) RailItem {
	leftGap, rightGap := determineGaps(width, self.width)

	self.addChild(newPath(x, y).h(leftGap))
	self.addChild(newPath(x+leftGap+self.width, y+self.height).h(rightGap))
	x += leftGap

	self.addChild(newPath(x, y).right(ConfigArcRadius))
	self.addChild(self.item.format(x+ConfigArcRadius, y, self.width-ConfigArcRadius*2))
	self.addChild(newPath(x+self.width-ConfigArcRadius, y+self.height).right(ConfigArcRadius))

	distanceFromY := max(ConfigArcRadius*2, self.item.getHeight()+
		self.item.getDown()+ConfigVerticalSeparation+self.rep.getUp())
	self.addChild(newPath(x+ConfigArcRadius, y).
		arc("nw").down(distanceFromY - ConfigArcRadius*2).
		arc("ws"))
	self.addChild(self.rep.format(x+ConfigArcRadius, y+distanceFromY, self.width-ConfigArcRadius*2))
	self.addChild(newPath(x+self.width-ConfigArcRadius, y+distanceFromY+self.rep.getHeight()).
		arc("se").up(distanceFromY - ConfigArcRadius*2 + self.rep.getHeight() - self.item.getHeight()).
		arc("en"))

	return self
}

type ZeroOrMoreOption struct {
	repeat *RailItem
	skip   *bool
}

func ZeroOrMoreSkip(skip bool) ZeroOrMoreOption         { return ZeroOrMoreOption{skip: &skip} }
func ZeroOrMoreRepeat(repeat RailItem) ZeroOrMoreOption { return ZeroOrMoreOption{repeat: &repeat} }

func ZeroOrMore(item RailItem, options ...ZeroOrMoreOption) RailItem {
	var (
		repeat RailItem
		skip   bool
	)
	for _, opt := range options {
		if opt.repeat != nil {
			repeat = *opt.repeat
		}
		if opt.skip != nil {
			skip = *opt.skip
		}
	}
	return Optional(OneOrMore(item, OneOrMoreRepeat(repeat)), OptionalSkip(skip))
}

type start struct {
	*diagramItem
}

func newStart() RailItem {
	di := newDiagramItem("path", nil)
	di.width = 20
	di.up = 10
	di.down = 10
	// TODO debug
	return &start{
		diagramItem: di,
	}
}

func (s *start) format(x, y, width float64) RailItem {
	s.attrs["d"] = fmt.Sprintf("M %v %v v 20 m 10 -20 v 20 m -10 -10 h 20.5", x, y-10)
	return s
}

type end struct {
	*diagramItem
}

func newEnd() RailItem {
	di := newDiagramItem("path", nil)
	di.width = 20
	di.up = 10
	di.down = 10
	// TODO debug
	return &end{
		diagramItem: di,
	}
}

func (self *end) format(x, y, width float64) RailItem {
	self.attrs["d"] = fmt.Sprintf("M %v %v h 20 m -10 -10 v 20 m 10 -20 v 20", x, y)
	return self
}

type terminal struct {
	*diagramItem
	text string
}

func Terminal(text string) RailItem {
	di := newDiagramItem("g", a{"class": "terminal"})
	di.width = float64(len(text))*ConfigCharacterAdvance + 20
	di.up = 11
	di.down = 11
	di.needsSpace = true
	return &terminal{
		diagramItem: di,
		text:        text,
	}
}

func (self *terminal) format(x, y, width float64) RailItem {
	leftGap, rightGap := determineGaps(width, self.width)

	self.addChild(newPath(x, y).h(leftGap))
	self.addChild(newPath(x+leftGap+self.width, y).h(rightGap))

	self.addChild(newDiagramItem("rect", a{
		"x":      fmt.Sprint(x + leftGap),
		"y":      fmt.Sprint(y - 11),
		"width":  fmt.Sprint(self.width),
		"height": fmt.Sprint(self.up + self.down),
		"rx":     "10",
		"ry":     "10",
	}))
	// TODO href
	self.addChild(newDiagramText("text", self.text, a{
		"x": fmt.Sprint(x + float64(int(width)/2)),
		"y": fmt.Sprint(y + 4),
	}))

	return self
}

type nonTerminal struct {
	*diagramItem
	text string
}

func NonTerminal(text string) RailItem {
	di := newDiagramItem("g", a{"class": "non-terminal"})
	di.width = float64(len(text))*ConfigCharacterAdvance + 20
	di.up = 11
	di.down = 11
	di.needsSpace = true
	return &nonTerminal{
		diagramItem: di,
		text:        text,
	}
}

func (self *nonTerminal) format(x, y, width float64) RailItem {
	leftGap, rightGap := determineGaps(width, self.width)

	self.addChild(newPath(x, y).h(leftGap))
	self.addChild(newPath(x+leftGap+self.width, y).h(rightGap))

	self.addChild(newDiagramItem("rect", a{
		"x":      fmt.Sprint(x + leftGap),
		"y":      fmt.Sprint(y - 11),
		"width":  fmt.Sprint(self.width),
		"height": fmt.Sprint(self.up + self.down),
	}))
	// TODO href
	self.addChild(newDiagramText("text", self.text, a{
		"x": fmt.Sprint(x + float64(int(width/2))),
		"y": fmt.Sprint(y + 4),
	}))

	return self
}

type comment struct {
	*diagramItem
	text string
}

func Comment(text string) RailItem {
	di := newDiagramItem("g", nil)
	di.width = float64(len(text))*7 + 10
	di.up = 11
	di.down = 11
	di.needsSpace = true
	return &comment{
		diagramItem: di,
		text:        text,
	}
}

func (self *comment) format(x, y, width float64) RailItem {
	leftGap, rightGap := determineGaps(width, self.width)

	self.addChild(newPath(x, y).h(leftGap))
	self.addChild(newPath(x+leftGap+self.width, y).h(rightGap))

	// TODO href
	self.addChild(newDiagramText("text", self.text, a{
		"x":     fmt.Sprint(x + float64(int(width)/2)),
		"y":     fmt.Sprint(y + 5),
		"class": "comment",
	}))

	return self
}

type skip struct {
	*diagramItem
}

func Skip() RailItem {
	di := newDiagramItem("g", nil)
	di.width = 0
	di.up = 0
	di.down = 0
	// TODO debug
	return &skip{
		diagramItem: di,
	}
}

func (self *skip) format(x, y, width float64) RailItem {
	self.addChild(newPath(x, y).right(width))
	return self
}
