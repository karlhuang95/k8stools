package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
)

// OutputData 输出通用数据（支持 csv/json/table）
func OutputData(headers []string, rows [][]string, format string) {
	switch format {
	case "table":
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader(headers)
		table.AppendBulk(rows)
		table.Render()
	case "json":
		var jsonList []map[string]string
		for _, row := range rows {
			entry := make(map[string]string)
			for i := range headers {
				entry[headers[i]] = row[i]
			}
			jsonList = append(jsonList, entry)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(jsonList); err != nil {
			fmt.Println("❌ JSON 输出失败:", err)
		}
	case "csv":
		writer := csv.NewWriter(os.Stdout)
		writer.Write(headers)
		writer.WriteAll(rows)
		writer.Flush()
	default:
		fmt.Printf("❌ 不支持的输出格式: %s (请使用 csv/json/table)\n", format)
	}
}
