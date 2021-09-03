// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/util/jsonpath"
)

func registerTableOutputFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("table", false, "Whether to display the returned output list as a table or not.")
	cmd.Flags().StringSlice("custom-columns", []string{}, "Custom columns for table output specified with jsonpath in form <column_name>:<jsonpath>. Example: --custom-columns=ID:.ID,State:.State,VPC:.ProvisionerMetadataKops.VPC")
}

func tableOutputEnabled(command *cobra.Command) (bool, []string) {
	outputToTable, _ := command.Flags().GetBool("table")
	customCols, _ := command.Flags().GetStringSlice("custom-columns")

	return outputToTable || len(customCols) > 0, customCols
}

func printTable(columnNames []string, values [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeader(columnNames)

	for _, v := range values {
		table.Append(v)
	}
	table.Render()
}

func prepareTableData(customCols []string, data []interface{}) ([]string, [][]string, error) {
	cc, err := parseCustomColumns(customCols)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse custom columns")
	}

	vals, err := makeValues(cc, data)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to make table values")
	}

	return cc.keys, vals, nil
}

type customColumns struct {
	keys             []string
	valueExpressions []*jsonpath.JSONPath
}

func parseCustomColumns(customCols []string) (customColumns, error) {
	columnKeys := make([]string, 0, len(customCols))
	columnExpressions := make([]*jsonpath.JSONPath, 0, len(customCols))

	for i, expr := range customCols {
		colSpec := strings.SplitN(expr, ":", 2)
		if len(colSpec) != 2 {
			return customColumns{}, fmt.Errorf("unexpected custom-columns spec: %s, expected <header>:<json-path-expr>", expr)
		}
		columnKeys = append(columnKeys, colSpec[0])

		spec, err := relaxedJSONPathExpression(colSpec[1])
		if err != nil {
			return customColumns{}, err
		}

		parser := jsonpath.New(fmt.Sprintf("column%d", i)).AllowMissingKeys(true)
		if err := parser.Parse(spec); err != nil {
			return customColumns{}, errors.Wrapf(err, "failed to parse jsonpath expression %q", spec)
		}

		columnExpressions = append(columnExpressions, parser)
	}

	return customColumns{keys: columnKeys, valueExpressions: columnExpressions}, nil
}

func makeValues(cc customColumns, data []interface{}) ([][]string, error) {
	rows := make([][]string, 0, len(data))
	for _, elem := range data {
		valRow, err := makeRow(cc.valueExpressions, elem)
		if err != nil {
			return nil, err
		}
		rows = append(rows, valRow)
	}
	return rows, nil
}

func makeRow(jpExpression []*jsonpath.JSONPath, data interface{}) ([]string, error) {
	vals := make([]string, 0, len(jpExpression))

	for _, exp := range jpExpression {
		values, err := exp.FindResults(data)
		if err != nil {
			return nil, errors.Wrap(err, "failed to find expression results for data")
		}

		var valuesStr []string
		if len(values) == 0 || len(values[0]) == 0 {
			valuesStr = append(valuesStr, "<none>")
		}

		for arrIx := range values {
			for valIx := range values[arrIx] {
				valuesStr = append(valuesStr, fmt.Sprintf("%v", values[arrIx][valIx].Interface()))
			}
		}
		vals = append(vals, strings.Join(valuesStr, ","))
	}

	return vals, nil
}

var jsonRegexp = regexp.MustCompile(`^\{\.?([^{}]+)\}$|^\.?([^{}]+)$`)

// relaxedJSONPathExpression attempts to be flexible with JSONPath expressions,
// it accepts following formats:
//	* {.ID}
//	* {ID}
//	* .ID
//	* ID
func relaxedJSONPathExpression(pathExpression string) (string, error) {
	if len(pathExpression) == 0 {
		return pathExpression, nil
	}
	submatches := jsonRegexp.FindStringSubmatch(pathExpression)
	if submatches == nil {
		return "", fmt.Errorf("unexpected path string, expected a 'name1.name2' or '.name1.name2' or '{name1.name2}' or '{.name1.name2}'")
	}
	if len(submatches) != 3 {
		return "", fmt.Errorf("unexpected submatch list: %v", submatches)
	}
	var fieldSpec string
	if len(submatches[1]) != 0 {
		fieldSpec = submatches[1]
	} else {
		fieldSpec = submatches[2]
	}
	return fmt.Sprintf("{.%s}", fieldSpec), nil
}
