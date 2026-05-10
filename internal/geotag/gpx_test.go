package geotag

import (
	"fmt"
	"testing"

	"github.com/qichengzx/coordtransform"
)

func TestName(t *testing.T) {
	lng := 113.281924
	lat := 23.201277
	wgsLng, wgsLat := coordtransform.GCJ02toWGS84(lng, lat)
	fmt.Println(wgsLat, wgsLng)
}

func TestGuessCoordSystem(t *testing.T) {
	// link GPX/workdoorout-TuesdayEveningWalk.gpx:4
	lng := 113.3979523
	lat := 23.1286279
	fmt.Println("ws WGS8")
	fmt.Println(fmt.Sprintf("%f,%f", lng, lat))
	toDMS := DecimalToDMS(lng, lat)
	fmt.Println(toDMS)
	wgsLng, wgsLat := coordtransform.WGS84toGCJ02(lng, lat)
	fmt.Println("ws WGS84toGCJ02")
	fmt.Println(fmt.Sprintf("%f,%f", wgsLng, wgsLat))
	toDMS = DecimalToDMS(wgsLng, wgsLat)
	fmt.Println(toDMS)

	fmt.Println("===========")
	// link GPX/2025-11-15 11 51 31.gpx:10
	lng = 113.281924
	lat = 23.201277
	fmt.Println("liagbulu WGS8")
	fmt.Println(fmt.Sprintf("%f,%f", lng, lat))
	toDMS = DecimalToDMS(lng, lat)
	fmt.Println(toDMS)
	wgsLng, wgsLat = coordtransform.WGS84toGCJ02(lng, lat)
	fmt.Println("liagbulu WGS84toGCJ02")
	fmt.Println(fmt.Sprintf("%f,%f", wgsLng, wgsLat))
	toDMS = DecimalToDMS(wgsLng, wgsLat)
	fmt.Println(toDMS)
	/*
		两步路导出的 GPX
		很可能是“标准 GPX”
		内部存 WGS84
		GPX 标准规范
	*/

	// 17/23.199408/113.283956
	lng = 113.283956
	lat = 23.199408
	fmt.Println("liagbulu WGS8")
	fmt.Println(fmt.Sprintf("%f,%f", lng, lat))
	toDMS = DecimalToDMS(lng, lat)
	fmt.Println(toDMS)
	wgsLng, wgsLat = coordtransform.WGS84toGCJ02(lng, lat)
	fmt.Println("liagbulu WGS84toGCJ02")
	fmt.Println(fmt.Sprintf("%f,%f", wgsLng, wgsLat))
	toDMS = DecimalToDMS(wgsLng, wgsLat)
	fmt.Println(toDMS)
}
