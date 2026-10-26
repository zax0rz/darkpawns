package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestShopTradeExpressionMatchesCStackSemantics(t *testing.T) {
	item := NewObjectInstance(&parser.Obj{Keywords: "gold ring", TypeFlag: 13, Cost: 20, ExtraFlags: [4]int{1 << 6}}, 1)
	for _, tc := range []struct {
		expression string
		want       bool
	}{
		{"", true},
		{"(absent)", true},
		{"^gold", true},
		{" gold", true},
		{"gold", true},
		{"gol", false},
		{"absent", false},
		{"MAGIC", true},
		{"glow", false},
		{"gold & MAGIC", true},
		{"gold * MAGIC", true},
		{"absent | gold", false},
		{"gold | absent", true},
		{"gold + absent", true},
		{"gold | MAGIC", false},
		{"gold & absent", false},
		{"absent & gold", false},
		{"absent ^", true},
		{"gold & (MAGIC)", true},
		{"gold & {MAGIC}", true},
		{"gold & [MAGIC]", true},
		{"gold )", false},
		{"gold ring", false},
	} {
		if got := shopEvaluateExpression(item, tc.expression); got != tc.want {
			t.Errorf("expr=%q got=%t want=%t", tc.expression, got, tc.want)
		}
	}
}

func TestShopTradeWithMatchesCRefusalOrder(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		kind, cost, charges, flags int
		types                      []int
		words                      []string
		want                       ShopTradeResult
	}{
		{"zero value", ITEM_WAND, 0, 0, 0, []int{ITEM_WAND}, nil, ShopTradeNotOK},
		{"nosell before dead", ITEM_WAND, 10, 0, 1 << 16, []int{ITEM_WAND}, nil, ShopTradeNotOK},
		{"wrong type before dead", ITEM_WAND, 10, 0, 0, []int{13}, nil, ShopTradeNotOK},
		{"dead wand before expression", ITEM_WAND, 10, 0, 0, []int{ITEM_WAND}, []string{"absent"}, ShopTradeDead},
		{"dead staff", ITEM_STAFF, 10, 0, 0, []int{ITEM_STAFF}, nil, ShopTradeDead},
		{"charged wand", ITEM_WAND, 10, 1, 0, []int{ITEM_WAND}, nil, ShopTradeOK},
		{"keyword refusal", 13, 10, 0, 0, []int{13}, []string{"silver"}, ShopTradeNotOK},
		{"later matching rule", 13, 10, 0, 0, []int{13, 13}, []string{"silver", "gold"}, ShopTradeOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			shop := &Shop{BuyTypes: tc.types, BuyWords: tc.words}
			item := NewObjectInstance(&parser.Obj{TypeFlag: tc.kind, Keywords: "gold ring", Cost: tc.cost, Values: [4]int{0, 0, tc.charges, 0}, ExtraFlags: [4]int{tc.flags}}, 1)
			if got := shop.TradeWith(item); got != tc.want {
				t.Errorf("trade=%d want=%d", got, tc.want)
			}
		})
	}
}
