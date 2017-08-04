// Copyright 2016 NDP Systèmes. All Rights Reserved.
// See LICENSE file for full licensing details.

package base

import (
	"fmt"
	"math"

	"github.com/hexya-erp/hexya/hexya/models"
	"github.com/hexya-erp/hexya/hexya/models/types"
	"github.com/hexya-erp/hexya/hexya/models/types/dates"
	"github.com/hexya-erp/hexya/hexya/tools/nbutils"
	"github.com/hexya-erp/hexya/pool"
)

func init() {
	currencyRateModel := pool.CurrencyRate().DeclareModel()
	currencyRateModel.AddDateTimeField("Name", models.SimpleFieldParams{String: "Date", Required: true, Index: true})
	currencyRateModel.AddFloatField("Rate", models.FloatFieldParams{Digits: nbutils.Digits{Precision: 12, Scale: 6},
		Help: "The rate of the currency to the currency of rate 1"})
	currencyRateModel.AddMany2OneField("Currency", models.ForeignKeyFieldParams{RelationModel: pool.Currency()})
	currencyRateModel.AddMany2OneField("Company", models.ForeignKeyFieldParams{RelationModel: pool.Company()})

	currencyModel := pool.Currency().DeclareModel()
	currencyModel.AddCharField("Name", models.StringFieldParams{String: "Currency", Help: "Currency Code [ISO 4217]", Size: 3,
		Unique: true})
	currencyModel.AddCharField("Symbol", models.StringFieldParams{Help: "Currency sign, to be used when printing amounts", Size: 4})
	currencyModel.AddFloatField("Rate", models.FloatFieldParams{String: "Current Rate",
		Help: "The rate of the currency to the currency of rate 1", Digits: nbutils.Digits{Precision: 12, Scale: 6},
		Compute: "ComputeCurrentRate"})
	currencyModel.AddOne2ManyField("Rates", models.ReverseFieldParams{RelationModel: pool.CurrencyRate(), ReverseFK: "Currency"})
	currencyModel.AddFloatField("Rounding", models.FloatFieldParams{String: "Rounding Factor", Digits: nbutils.Digits{Precision: 12,
		Scale: 6}})
	currencyModel.AddIntegerField("DecimalPlaces", models.SimpleFieldParams{GoType: new(int8), Compute: "ComputeDecimalPlaces",
		Depends: []string{"Rounding"}})
	currencyModel.AddBooleanField("Active", models.SimpleFieldParams{})
	currencyModel.AddSelectionField("Position", models.SelectionFieldParams{Selection: types.Selection{"after": "After Amount", "before": "Before Amount"},
		String: "Symbol Position", Help: "Determines where the currency symbol should be placed after or before the amount."})
	currencyModel.AddDateField("Date", models.SimpleFieldParams{Compute: "ComputeDate"})

	currencyModel.Methods().ComputeCurrentRate().DeclareMethod(
		`ComputeCurrentRate returns the current rate of this currency.
		 If a 'date' key (type DateTime) is given in the context, then it is used to compute the rate,
		 otherwise now is used.`,
		func(rs pool.CurrencySet) (*pool.CurrencyData, []models.FieldNamer) {
			date := dates.Now()
			if rs.Env().Context().HasKey("date") {
				date = rs.Env().Context().GetDateTime("date")
			}
			company := pool.User().NewSet(rs.Env()).GetCompany()
			if rs.Env().Context().HasKey("company_id") {
				company = pool.Company().Browse(rs.Env(), []int64{rs.Env().Context().GetInteger("company_id")})
			}
			rate := pool.CurrencyRate().Search(rs.Env(),
				pool.CurrencyRate().Currency().Equals(rs).
					And().Name().LowerOrEqual(date).
					AndCond(
						pool.CurrencyRate().Company().IsNull().
							Or().Company().Equals(company))).
				OrderBy("Company", "Name desc").
				Limit(1)
			res := rate.Rate()
			if res == 0 {
				res = 1.0
			}
			return &pool.CurrencyData{Rate: res}, []models.FieldNamer{pool.Currency().Rate()}
		})

	currencyModel.Methods().ComputeDecimalPlaces().DeclareMethod(
		`ComputeDecimalPlaces returns the decimal place from the currency's rounding`,
		func(rs pool.CurrencySet) (*pool.CurrencyData, []models.FieldNamer) {
			var dp int8
			if rs.Rounding() > 0 && rs.Rounding() < 1 {
				dp = int8(math.Ceil(math.Log10(1 / rs.Rounding())))
			}
			return &pool.CurrencyData{DecimalPlaces: dp}, []models.FieldNamer{pool.Currency().DecimalPlaces()}
		})

	currencyModel.Methods().ComputeDate().DeclareMethod(
		`ComputeDate returns the date of the last rate of this currency`,
		func(rs pool.CurrencySet) (*pool.CurrencyData, []models.FieldNamer) {
			var lastDate dates.Date
			if rateLength := len(rs.Rates().Records()); rateLength > 0 {
				lastDate = rs.Rates().Records()[rateLength-1].Name().ToDate()
			}
			return &pool.CurrencyData{Date: lastDate}, []models.FieldNamer{pool.Currency().Date()}
		})

	currencyModel.Methods().Round().DeclareMethod(
		`Round returns the given amount rounded according to this currency rounding rules`,
		func(rs pool.CurrencySet, amount float64) float64 {
			return nbutils.Round(amount, nbutils.Digits{Scale: rs.DecimalPlaces()})
		})

	currencyModel.Methods().CompareAmounts().DeclareMethod(
		`CompareAmounts compares 'amount1' and 'amount2' after rounding them according
		 to the given currency's precision. The returned values are per the following table:

		     value1 > value2 : true, false
    	     value1 == value2: false, true
    	     value1 < value2 : false, false

		 An amount is considered lower/greater than another amount if their rounded
         value is different. This is not the same as having a non-zero difference!

         For example 1.432 and 1.431 are equal at 2 digits precision,
         so this method would return 0.
         However 0.006 and 0.002 are considered different (returns 1) because
         they respectively round to 0.01 and 0.0, even though 0.006-0.002 = 0.004
         which would be considered zero at 2 digits precision.`,
		func(rs pool.CurrencySet, amount1, amount2 float64) (greater, equal bool) {
			return nbutils.Compare(amount1, amount2, nbutils.Digits{Scale: rs.DecimalPlaces()})
		})

	currencyModel.Methods().IsZero().DeclareMethod(
		`IsZero returns true if 'amount' is small enough to be treated as
		zero according to current currency's rounding rules.

		Warning: IsZero(amount1-amount2) is not always equivalent to
		CompareAmomuts(amount1,amount2) == _, true, as the former will
		round after computing the difference, while the latter will round
		before, giving different results for e.g. 0.006 and 0.002 at 2
		digits precision.`,
		func(rs pool.CurrencySet, amount float64) bool {
			return nbutils.IsZero(amount, nbutils.Digits{Scale: rs.DecimalPlaces()})
		})

	currencyModel.Methods().GetConversionRateTo().DeclareMethod(
		`GetConversionRateTo returns the conversion rate from this currency to 'target' currency`,
		func(rs pool.CurrencySet, target pool.CurrencySet) float64 {
			return target.WithEnv(rs.Env()).Rate() / rs.Rate()
		})

	currencyModel.Methods().Compute().DeclareMethod(
		`Compute converts 'amount' from this currency to 'targetCurrency'.
		 The result is rounded to the 'target' currency if 'round' is true.`,
		func(rs pool.CurrencySet, amount float64, target pool.CurrencySet, round bool) float64 {
			if rs.Equals(target) {
				if round {
					return rs.Round(amount)
				}
				return amount
			}
			res := amount * rs.GetConversionRateTo(target)
			if round {
				return target.Round(res)
			}
			return res
		})

	currencyModel.Methods().GetFormatCurrenciesJsFunction().DeclareMethod(
		`GetFormatCurrenciesJsFunction returns a string that can be used to instanciate a javascript
		function that formats numbers as currencies.

		That function expects the number as first parameter	and the currency id as second parameter.
		If the currency id parameter is false or undefined, the	company currency is used.`,
		func(rs pool.CurrencySet) string {
			companyCurrency := pool.User().Browse(rs.Env(), []int64{rs.Env().Uid()}).Company().Currency()
			var function string
			for _, currency := range pool.Currency().NewSet(rs.Env()).FetchAll().Records() {
				symbol := currency.Symbol()
				if symbol == "" {
					symbol = currency.Name()
				}
				formatNumberStr := fmt.Sprintf("hexyaerp.web.format_value(arguments[0], {type: 'float', digits: [69,%d]}, 0.00)", currency.DecimalPlaces())
				returnStr := fmt.Sprintf("return %s + '\\xA0' + %s;", formatNumberStr, symbol)
				if currency.Position() == "before" {
					returnStr = fmt.Sprintf("return %s + '\\xA0' + %s;", symbol, formatNumberStr)
				}
				function += fmt.Sprintf("if (arguments[1] === %s) { %s }", currency.ID(), returnStr)
				if currency.Equals(companyCurrency) {
					companyCurrentFormat := returnStr
					function = fmt.Sprintf("if (arguments[1] === false || arguments[1] === undefined) { %s }%s", companyCurrentFormat, function)
				}
			}
			return function
		})

}
