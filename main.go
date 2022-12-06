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

type EtsyData map[string]string
type MoneyData map[string]int
type LowStockData map[string]int

// Ls = Low stock
const (
	EtsySKUColumn       int = 23
	EtsyTitleColumn     int = 0
	MoneySKUColumn      int = 3
	MoneyQuantityColumn int = 4
	LsSKUColumn         int = 0
	LsQuantityColumn    int = 1

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

func reportLowStock(ed EtsyData, md MoneyData) LowStockData {
	lsd := make(LowStockData)
	var stock_sub_0, stock_sub_10, stock_sub_50 []string

	for sku := range ed {
		q, in_money := md[sku]

		// skip SKU that isn't in money_data
		if !in_money {
			continue
		}

		str := fmt.Sprintf("%s,%d", sku, q)
		// check for different levels of shortage
		if q < 0 {
			stock_sub_0 = append(stock_sub_0, str)
			lsd[sku] = q
		} else if q < 10 {
			stock_sub_10 = append(stock_sub_10, str)
			lsd[sku] = q
		} else if q < 50 {
			stock_sub_50 = append(stock_sub_50, str)
			lsd[sku] = q
		}
	}

	// combine all reports into "stock_sub_all"
	low_stock_all := append(stock_sub_0, stock_sub_10...)
	low_stock_all = append(low_stock_all, stock_sub_50...)

	if len(lsd) <= 0 {
		fmt.Println("No low stock found")
		return lsd
	}

	// write to all the files
	const header = "SKU,QUANTITY"

	//writeToPwd(filenameLowStockSub0, stock_sub_0, header)
	//writeToPwd(filenameLowStockSub10, stock_sub_10, header)
	//writeToPwd(filenameLowStockSub50, stock_sub_50, header)

	writeToReportFolder(filenameLowStockSub0, stock_sub_0, header)
	writeToReportFolder(filenameLowStockSub10, stock_sub_10, header)
	writeToReportFolder(filenameLowStockSub50, stock_sub_50, header)

	writeToPwd(filenameLowStock, low_stock_all, header)
	writeToReportFolder(filenameLowStock, low_stock_all, header)

	return lsd
}

func reportNewLowStock(olsd LowStockData, lsd LowStockData) {
	var new_low_stock []string

	// for every new low_stock SKUs
	for sku := range lsd {
		q, in_old := olsd[sku]

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

		writeToReportFolder(filename, new_low_stock, header)

		fmt.Println("Changed file: " + filename)
	} else {
		fmt.Println("No new low stock found")
	}

}

func reportWrongSKU(ed EtsyData, md MoneyData) {
	var wrong_sku []string

	for sku := range ed {
		_, in_money := md[sku]

		// skip SKU that isn't in money_data
		if !in_money {
			title := strings.TrimSpace(ed[sku])
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
		writeToReportFolder(filename, wrong_sku, header)

		fmt.Println("Generated: " + filename)
	} else {
		fmt.Println("No wrong SKUs found")
	}
}

func reportRestock(olsd LowStockData, md MoneyData) {
	var restocks []string

	for sku, old_q := range olsd {
		new_q, in_money := md[sku]
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

		writeToReportFolder(filename, restocks, header)

		fmt.Println("Generated: " + filename)
	} else {
		fmt.Println("No restocks found")
	}

}

func writeToPwd(fn string, lns []string, h string) {
	pwd := getPwd()

	fullFilePath := filepath.Join(pwd, fn)

	f, err := os.Create(fullFilePath)
	check(err)

	defer f.Close()

	w := bufio.NewWriter(f)

	_, err = w.WriteString(strings.Join(append([]string{h}, lns...), "\n"))
	check(err)

	w.Flush()
}

func writeToReportFolder(fn string, lns []string, h string) {
	pwd := getPwd()

	dateFolderName := generateCurrentDateFolderName()
	reportFilepath := filepath.Join(pwd, "reports", dateFolderName)

	_, err := os.Stat(reportFilepath)
	if !os.IsNotExist(err) {
		check(err)
	}

	err = os.MkdirAll(reportFilepath, os.ModePerm)
	check(err)

	f, err := os.Create(filepath.Join(reportFilepath, fn))
	check(err)

	defer f.Close()

	w := bufio.NewWriter(f)

	_, err = w.WriteString(strings.Join(append([]string{h}, lns...), "\n"))
	check(err)

	w.Flush()
}

func generateCurrentDateFolderName() string {
	n := time.Now().Local()
	return fmt.Sprintf("%d%d%d_%02d%02d", n.Year(), n.Day(), n.Month(), n.Hour(), n.Minute())
}

// ==== LOADERS ====

func loadEtsyData(fp string) EtsyData {
	f, err := os.Open(fp)
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

		if strings.Contains(row[EtsySKUColumn], ",") {
			// split into SKUs
			skus := strings.Split(row[EtsySKUColumn], ",")

			// add each SKU to map
			for _, sku := range skus {
				title := row[EtsyTitleColumn]
				etsy_listings[sku] = title
			}

		} else {
			// add SKU to map
			sku := row[EtsySKUColumn]
			title := row[EtsyTitleColumn]
			etsy_listings[sku] = title[:1]
		}
	}

	return etsy_listings
}

func loadMoneyData(fp string) MoneyData {
	f, err := os.Open(fp)
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
		sku := row[MoneySKUColumn]
		// load quantity as string
		quantityString := row[MoneyQuantityColumn]
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

func loadLowStockData(fp string) LowStockData {
	f, err := os.Open(fp)
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
		sku := row[LsSKUColumn]
		// load quantity as string
		quantityString := row[LsQuantityColumn]
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
