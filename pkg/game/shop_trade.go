package game

import (
	"strings"
	"unicode"

	"github.com/zax0rz/darkpawns/pkg/olc"
)

// ShopTradeResult follows src/shop.h:82-84, shared by player/NPC selling
// and valuation. Dead wands have their own keeper refusal.
type ShopTradeResult int

const (
	ShopTradeDead ShopTradeResult = iota
	ShopTradeNotOK
	ShopTradeOK
)

// TradeWith ports trade_with (src/shop.c:268-289), including the optional
// keyword/extra-flag expression paired with each accepted object type.
func (s *Shop) TradeWith(item *ObjectInstance) ShopTradeResult {
	if item.GetCost() < 1 || item.HasExtraFlag(0, 16) {
		return ShopTradeNotOK
	}
	for i, t := range s.BuyTypes {
		if t != item.GetTypeFlag() {
			continue
		}
		if (t == ITEM_WAND || t == ITEM_STAFF) && item.GetValue(2) == 0 {
			return ShopTradeDead
		}
		expression := ""
		if i < len(s.BuyWords) {
			expression = s.BuyWords[i]
		}
		if shopEvaluateExpression(item, expression) {
			return ShopTradeOK
		}
	}
	return ShopTradeNotOK
}

// shopEvaluateExpression follows src/shop.c:143-264, including C's initial
// nonalphabetic bypass and the short-circuit pop order in evaluate_operation.
func shopEvaluateExpression(item *ObjectInstance, expression string) bool {
	if expression == "" {
		return true
	}
	first := expression[0]
	if (first < 'A' || first > 'Z') && (first < 'a' || first > 'z') {
		return true
	}
	operators := []int{}
	values := []bool{}
	popValue := func() bool {
		if len(values) == 0 {
			return false
		}
		n := len(values) - 1
		v := values[n]
		values = values[:n]
		return v
	}
	top := func() int {
		if len(operators) == 0 {
			return -1
		}
		return operators[len(operators)-1]
	}
	popOperator := func() int {
		if len(operators) == 0 {
			return -1
		}
		n := len(operators) - 1
		v := operators[n]
		operators = operators[:n]
		return v
	}
	evaluate := func() {
		switch popOperator() {
		case 4:
			value := !popValue()
			values = append(values, value)
		case 3:
			right := popValue()
			if right {
				right = popValue()
			}
			values = append(values, right)
		case 2:
			right := popValue()
			if !right {
				right = popValue()
			}
			values = append(values, right)
		}
	}
	operation := func(c byte) int {
		for i, tokens := range []string{"[({", "])}", "|+", "&*", "^'"} {
			if strings.ContainsRune(tokens, rune(c)) {
				return i
			}
		}
		return -1
	}
	for i := 0; i < len(expression); {
		if unicode.IsSpace(rune(expression[i])) {
			i++
			continue
		}
		op := operation(expression[i])
		if op < 0 {
			start := i
			for i < len(expression) && !unicode.IsSpace(rune(expression[i])) && operation(expression[i]) < 0 {
				i++
			}
			name := expression[start:i]
			value := false
			flag := false
			for bit, label := range olc.ItemExtraFlagNames {
				if strings.EqualFold(name, label) {
					value = item.HasExtraFlag(bit/32, bit%32)
					flag = true
					break
				}
			}
			if !flag {
				value = isCName(name, item.GetKeywords())
			}
			values = append(values, value)
		} else {
			if op != 0 {
				for top() > op {
					evaluate()
				}
			}
			if op == 1 {
				if popOperator() != 0 {
					return false
				}
			} else {
				operators = append(operators, op)
			}
			i++
		}
	}
	for top() != -1 {
		evaluate()
	}
	result := popValue()
	return len(values) == 0 && result
}
