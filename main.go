package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	etsyColumnSKU       int = 23
	etsyColumnTitle     int = 0
	moneyColumnSKU      int = 3
	moneyColumnQuantity int = 4
	olsColumnSKU        int = 0
	olsColumnQuantity   int = 1

	filenameLowStock      string = "low_stock.csv"
	filenameLowStockSub0  string = "low_stock_sub0.csv"
	filenameLowStockSub10 string = "low_stock_sub10.csv"
	filenameLowStockSub50 string = "low_stock_sub50.csv"
	filenameWrongSKU      string = "wrong_sku.csv"
	filenameRestock       string = "restocked.csv"
	filenameLowStockNew   string = "low_stock_new.csv"
)

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func main() {
	// filepaths from flags
	etsyFilenamePtr := flag.String("etsy", "etsy.csv", "filename")
	moneyFilenamePtr := flag.String("money", "sklad.csv", "filename")
	olsFilenamePtr := flag.String("ols", "", "old low stock")

	flag.Parse()

	pwd := getPwd()

	etsyFilepath := filepath.Join(pwd, *etsyFilenamePtr)
	moneyFilepath := filepath.Join(pwd, *moneyFilenamePtr)

	fmt.Println("Etsy CSV filepath:", etsyFilepath)
	fmt.Println("Money CSV filepath:", moneyFilepath)

	// load Etsy data to map
	etsy_data := loadEtsyData(etsyFilepath)
	// load Money data to map
	money_data := loadMoneyData(moneyFilepath)

	// report low stock to 4 files (all, sub0, sub10, sub50)
	// rewrites current low_stock.csv
	low_stock := reportLowStock(etsy_data, money_data)

	// report SKUs that are on Etsy but not in Money
	// to file wrong_sku.csv
	reportWrongSKU(etsy_data, money_data)

	// if user set path to old low_stock,
	// check for restock and new low_stock
	if *olsFilenamePtr != "" {
		olsFilepath := filepath.Join(pwd, *olsFilenamePtr)

		fmt.Println("Old low stock CSV filepath:", olsFilepath)
		ols_data := loadLowStockData(olsFilepath)

		reportRestock(ols_data, money_data)
		reportNewLowStock(ols_data, low_stock)
	}

}

func getPwd() string {
	ex, err := os.Executable()
	check(err)
	fmt.Println(ex)
	return filepath.Dir(ex)
}

func reportLowStock(etsy_data map[string]string, money_data map[string]int) map[string]int {
	low_stock := make(map[string]int)
	var stock_sub_0, stock_sub_10, stock_sub_50 []string

	for sku := range etsy_data {
		q, in_money := money_data[sku]

		// skip SKU that isn't in money_data
		if !in_money {
			continue
		}

		str := fmt.Sprintf("%s,%d", sku, q)
		// check for different levels of shortage
		if q < 0 {
			stock_sub_0 = append(stock_sub_0, str)
			low_stock[sku] = q
		} else if q < 10 {
			stock_sub_10 = append(stock_sub_10, str)
			low_stock[sku] = q
		} else if q < 50 {
			stock_sub_50 = append(stock_sub_50, str)
			low_stock[sku] = q
		}
	}

	// all reports combined
	stock_sub_all := append(stock_sub_0, stock_sub_10...)
	stock_sub_all = append(stock_sub_all, stock_sub_50...)

	// write to all the files
	const header = "SKU,QUANTITY"
	writeToPwd(filenameLowStockSub0, stock_sub_0, header)
	writeToPwd(filenameLowStockSub10, stock_sub_10, header)
	writeToPwd(filenameLowStockSub50, stock_sub_50, header)

	writeToReportFolder(filenameLowStockSub0, stock_sub_0, header)
	writeToReportFolder(filenameLowStockSub10, stock_sub_10, header)
	writeToReportFolder(filenameLowStockSub50, stock_sub_50, header)

	//now := time.Now()
	// TODO: filename := fmt.Sprintf("low_stock%d%d.csv", now.Day(), now.Month())
	writeToPwd(filenameLowStock, stock_sub_all, header)

	return low_stock
}

func reportNewLowStock(ols_data map[string]int, low_stock_data map[string]int) {
	var new_low_stock []string

	// for every new low_stock SKUs
	for sku := range low_stock_data {
		q, in_old := ols_data[sku]

		// skip SKU that is also in the old low_stock
		if in_old {
			continue
		}

		str := fmt.Sprintf("%s,%d", sku, q)
		new_low_stock = append(new_low_stock, str)

	}

	if len(new_low_stock) > 0 {
		const header string = "SKU,QUANTITY"
		const filename string = filenameLowStockNew

		writeToPwd(filename, new_low_stock, header)
		fmt.Println("Changed file: " + filename)
	} else {
		fmt.Println("No new low stock found")
	}

}

func reportWrongSKU(etsy_data map[string]string, money_data map[string]int) {
	var wrong_sku []string

	for sku := range etsy_data {
		_, in_money := money_data[sku]

		// skip SKU that isn't in money_data
		if !in_money {
			title := strings.TrimSpace(etsy_data[sku])
			if len(title) > 60 {
				title = title[:60]
			}
			str := fmt.Sprintf("%s,\"%s\"", sku, title)
			wrong_sku = append(wrong_sku, str)
		}
	}

	if len(wrong_sku) > 0 {
		const header string = "SKU,TITLE"
		const filename string = filenameWrongSKU
		writeToPwd(filename, wrong_sku, header)
		writeToReportFolder(filename, wrong_sku, header)

		fmt.Println("Changed file: " + filename)
	} else {
		fmt.Println("No wrong SKUs found")
	}
}

func reportRestock(ols_data map[string]int, money_data map[string]int) {
	var restocks []string

	for sku, old_q := range ols_data {
		new_q, in_money := money_data[sku]
		if !in_money {
			continue
		}
		if new_q > old_q && new_q >= 50 {
			str := fmt.Sprintf("%s,%d,%d", sku, old_q, new_q)
			restocks = append(restocks, str)
		}
	}

	if len(restocks) > 0 {
		const header = "SKU,OLD QUANTITY,NEW QUANTITY"
		const filename string = filenameRestock
		writeToPwd(filename, restocks, header)
		writeToPwd(filename, restocks, header)

		fmt.Println("Changed file: " + filename)

	} else {
		fmt.Println("No restocks found")
	}

}

func writeToPwd(filename string, lines []string, header string) {
	pwd := getPwd()

	fullFilePath := filepath.Join(pwd, filename)

	f, err := os.Create(fullFilePath)
	check(err)

	defer f.Close()

	w := bufio.NewWriter(f)

	_, err = w.WriteString(strings.Join(append([]string{header}, lines...), "\n"))
	check(err)

	w.Flush()
}

func writeToReportFolder(filename string, lines []string, header string) {
	pwd := getPwd()

	n := time.Now().Local()
	dateFolderName := fmt.Sprintf("%d%d%d_%d%d", n.Year(), n.Day(), n.Month(), n.Hour(), n.Minute())
	reportFilepath := filepath.Join(pwd, "reports", dateFolderName)

	_, err := os.Stat(reportFilepath)
	if !os.IsNotExist(err) {
		check(err)
	}

	err = os.MkdirAll(reportFilepath, os.ModePerm)
	check(err)

	f, err := os.Create(filepath.Join(reportFilepath, filename))
	check(err)

	defer f.Close()

	w := bufio.NewWriter(f)

	_, err = w.WriteString(strings.Join(append([]string{header}, lines...), "\n"))
	check(err)

	w.Flush()
}

// ==== LOADERS ====

func loadEtsyData(filepath string) map[string]string {
	f, err := os.Open(filepath)
	check(err)

	r := csv.NewReader(bufio.NewReader(f))

	// reads the first line with column names
	if _, err := r.Read(); err != nil {
		log.Fatal(err)
	}

	etsy_listings := make(map[string]string)

	for {
		row, err := r.Read()

		if err == io.EOF {
			break
		}

		check(err)

		if strings.Contains(row[etsyColumnSKU], ",") {
			// split into SKUs
			skus := strings.Split(row[etsyColumnSKU], ",")

			// add each SKU to map
			for _, sku := range skus {
				title := row[etsyColumnTitle]
				etsy_listings[sku] = title
			}

		} else {
			// add SKU to map
			sku := row[etsyColumnSKU]
			title := row[etsyColumnTitle]
			etsy_listings[sku] = title[:1]
		}
	}

	return etsy_listings
}

func loadMoneyData(filepath string) map[string]int {
	f, err := os.Open(filepath)
	check(err)

	r := csv.NewReader(bufio.NewReader(f))

	// reads the first line with column names
	_, err = r.Read()
	check(err)

	money_listings := make(map[string]int)

	for {
		row, err := r.Read()

		if err == io.EOF {
			break
		}

		check(err)

		// get current SKU
		sku := row[moneyColumnSKU]
		// load quantity as string
		quantityString := row[moneyColumnQuantity]
		// replace comma by period
		quantityString = strings.ReplaceAll(quantityString, ",", ".")
		// convert string to float
		quantityFloat, err := strconv.ParseFloat(quantityString, 32)
		check(err)
		// add SKU with quantity as int to map
		money_listings[sku] = int(quantityFloat)
	}

	return money_listings
}

func loadLowStockData(filepath string) map[string]int {
	f, err := os.Open(filepath)
	check(err)

	r := csv.NewReader(bufio.NewReader(f))

	// reads the first line with column names
	_, err = r.Read()
	check(err)

	ols_listings := make(map[string]int)

	for {
		row, err := r.Read()

		if err == io.EOF {
			break
		}

		check(err)

		// get current SKU
		sku := row[olsColumnSKU]
		// load quantity as string
		quantityString := row[olsColumnQuantity]
		// replace comma by period
		quantityString = strings.ReplaceAll(quantityString, ",", ".")
		// convert string to float
		quantityFloat, err := strconv.ParseFloat(quantityString, 32)
		check(err)
		// add SKU with quantity as int to map
		ols_listings[sku] = int(quantityFloat)
	}

	return ols_listings
}
