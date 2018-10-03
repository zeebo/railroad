package railroad

import (
	"io"
	"io/ioutil"
	"testing"
)

func add(name string, diagram io.WriterTo) {
	diagram.WriteTo(ioutil.Discard)
}

func TestBasic(t *testing.T) {
	add(`comment`, Diagram(
		Text(`/*`),
		ZeroOrMore(
			NonTerminal(`anything but * followed by /`)),
		Text(`*/`)))

	add(`newline`, Diagram(Choice(0, Text(`\n`), Text(`\r\n`), Text(`\r`), Text(`\f`))))

	add(`whitespace`, Diagram(Choice(
		0, Text(`space`), Text(`\t`), NonTerminal(`newline`))))

	add(`hex digit`, Diagram(NonTerminal(`0-9 a-f or A-F`)))

	add(`escape`, Diagram(
		Text(`\`), Choice(0,
			NonTerminal(`not newline or hex digit`),
			Sequence(
				OneOrMore(NonTerminal(`hex digit`), OneOrMoreRepeat(Comment(`1-6 times`))),
				Optional(NonTerminal(`whitespace`), OptionalSkip(true))))))

	add(`<whitespace-token>`, Diagram(OneOrMore(NonTerminal(`whitespace`))))

	add(`ws*`, Diagram(ZeroOrMore(NonTerminal(`<whitespace-token>`))))

	add(`<ident-token>`, Diagram(
		Choice(0, Skip(), Text(`-`)),
		Choice(0, NonTerminal(`a-z A-Z _ or non-ASCII`), NonTerminal(`escape`)),
		ZeroOrMore(
			Choice(0, NonTerminal(`a-z A-Z 0-9 _ - or non-ASCII`), NonTerminal(`escape`)))))

	add(`<function-token>`, Diagram(
		NonTerminal(`<ident-token>`), Text(`(`)))

	add(`<at-keyword-token>`, Diagram(
		Text(`@`), NonTerminal(`<ident-token>`)))

	add(`<hash-token>`, Diagram(
		Text(`#`), OneOrMore(Choice(0,
			NonTerminal(`a-z A-Z 0-9 _ - or non-ASCII`),
			NonTerminal(`escape`)))))

	add(`<string-token>`, Diagram(
		Choice(0,
			Sequence(
				Text(`"`),
				ZeroOrMore(
					Choice(0,
						NonTerminal(`not " \ or newline`),
						NonTerminal(`escape`),
						Sequence(Text(`\`), NonTerminal(`newline`)))),
				Text(`"`)),
			Sequence(
				Text("'"),
				ZeroOrMore(
					Choice(0,
						NonTerminal("not ' \\ or newline"),
						NonTerminal(`escape`),
						Sequence(Text(`\`), NonTerminal(`newline`)))),
				Text("'")))))

	add(`<url-token>`, Diagram(
		NonTerminal(`<ident-token "url">`),
		Text(`(`),
		NonTerminal(`ws*`),
		Optional(Sequence(
			Choice(0, NonTerminal(`url-unquoted`), NonTerminal(`STRING`)),
			NonTerminal(`ws*`),
		)),
		Text(`)`)))

	add(`url-unquoted`, Diagram(OneOrMore(
		Choice(0,
			NonTerminal("not \" ' ( ) \\ whitespace or non-printable"),
			NonTerminal(`escape`)))))

	add(`<number-token>`, Diagram(
		Choice(1, Text(`+`), Skip(), Text(`-`)),
		Choice(0,
			Sequence(
				OneOrMore(NonTerminal(`digit`)),
				Text(`.`),
				OneOrMore(NonTerminal(`digit`))),
			OneOrMore(NonTerminal(`digit`)),
			Sequence(
				Text(`.`),
				OneOrMore(NonTerminal(`digit`)))),
		Choice(0,
			Skip(),
			Sequence(
				Choice(0, Text(`e`), Text(`E`)),
				Choice(1, Text(`+`), Skip(), Text(`-`)),
				OneOrMore(NonTerminal(`digit`))))))

	add(`<dimension-token>`, Diagram(
		NonTerminal(`<number-token>`), NonTerminal(`<ident-token>`)))

	add(`<percentage-token>`, Diagram(
		NonTerminal(`<number-token>`), Text(`%`)))

	add(`<unicode-range-token>`, Diagram(
		Choice(0, Text(`U`), Text(`u`)),
		Text(`+`),
		Choice(0,
			Sequence(OneOrMore(NonTerminal(`hex digit`), OneOrMoreRepeat(Comment(`1-6 times`)))),
			Sequence(
				ZeroOrMore(NonTerminal(`hex digit`), ZeroOrMoreRepeat(Comment(`1-5 times`))),
				OneOrMore(Terminal(`?`), OneOrMoreRepeat(Comment(`1 to (6 - digits) times`)))),
			Sequence(
				OneOrMore(NonTerminal(`hex digit`), OneOrMoreRepeat(Comment(`1-6 times`))),
				Text(`-`),
				OneOrMore(NonTerminal(`hex digit`), OneOrMoreRepeat(Comment(`1-6 times`)))))))

	add(`<include-match-token>`, Diagram(Text(`~=`)))

	add(`<dash-match-token>`, Diagram(Text(`|=`)))

	add(`<prefix-match-token>`, Diagram(Text(`^=`)))

	add(`<suffix-match-token>`, Diagram(Text(`$=`)))

	add(`<substring-match-token>`, Diagram(Text(`*=`)))

	add(`<column-token>`, Diagram(Text(`||`)))

	add(`<CDO-token>`, Diagram(Text(`<`+`!--`)))

	add(`<CDC-token>`, Diagram(Text(`-`+`->`)))

	add(`Stylesheet`, Diagram(ZeroOrMore(Choice(3,
		NonTerminal(`<CDO-token>`), NonTerminal(`<CDC-token>`), NonTerminal(`<whitespace-token>`),
		NonTerminal(`Qualified rule`), NonTerminal(`At-rule`)))))

	add(`Rule list`, Diagram(ZeroOrMore(Choice(1,
		NonTerminal(`<whitespace-token>`), NonTerminal(`Qualified rule`), NonTerminal(`At-rule`)))))

	add(`At-rule`, Diagram(
		NonTerminal(`<at-keyword-token>`), ZeroOrMore(NonTerminal(`Component value`)),
		Choice(0, NonTerminal(`{} block`), Text(`;`))))

	add(`Qualified rule`, Diagram(
		ZeroOrMore(NonTerminal(`Component value`)),
		NonTerminal(`{} block`)))

	add(`Declaration list`, Diagram(
		NonTerminal(`ws*`),
		Choice(0,
			Sequence(
				Optional(NonTerminal(`Declaration`)),
				Optional(Sequence(Text(`;`), NonTerminal(`Declaration list`)))),
			Sequence(
				NonTerminal(`At-rule`),
				NonTerminal(`Declaration list`)))))

	add(`Declaration`, Diagram(
		NonTerminal(`<ident-token>`), NonTerminal(`ws*`), Text(`:`),
		ZeroOrMore(NonTerminal(`Component value`)), Optional(NonTerminal(`!important`))))

	add(`!important`, Diagram(
		Text(`!`), NonTerminal(`ws*`), NonTerminal(`<ident-token "important">`), NonTerminal(`ws*`)))

	add(`Component value`, Diagram(Choice(0,
		NonTerminal(`Preserved token`),
		NonTerminal(`{} block`),
		NonTerminal(`() block`),
		NonTerminal(`[] block`),
		NonTerminal(`Function block`))))

	add(`{} block`, Diagram(Text(`{`), ZeroOrMore(NonTerminal(`Component value`)), Text(`}`)))
	add(`() block`, Diagram(Text(`(`), ZeroOrMore(NonTerminal(`Component value`)), Text(`)`)))
	add(`[] block`, Diagram(Text(`[`), ZeroOrMore(NonTerminal(`Component value`)), Text(`]`)))

	add(`Function block`, Diagram(
		NonTerminal(`<function-token>`),
		ZeroOrMore(NonTerminal(`Component value`)),
		Text(`)`)))
}
