package geodata_test

import (
	"fmt"

	"github.com/vincentchyu/photools/pkg/geodata"
)

func ExampleGetCountryNameZH() {
	fmt.Println(geodata.GetCountryNameZH("CN"))
	fmt.Println(geodata.GetCountryNameZH("US"))
	fmt.Println(geodata.GetCountryNameZH("JP"))

	// Output:
	// 中国
	// 美国
	// 日本
}

func ExampleGetAdmin1NameZH() {
	// GeoNames FIPS 10-4 省级代码
	fmt.Println(geodata.GetAdmin1NameZH("13"))
	fmt.Println(geodata.GetAdmin1NameZH("31"))

	// Output:
	// 新疆维吾尔自治区
	// 海南省
}

func ExampleGetAdmin2NameZH() {
	// 中国地级市 GB/T 2260 代码
	fmt.Println(geodata.GetAdmin2NameZH("6531"))

	// Output:
	// 喀什地区
}
