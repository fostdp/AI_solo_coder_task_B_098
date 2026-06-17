package analysis

import (
	"beacon-system/models"
	"math"
)

const (
	EarthRadius        = 6371.0
	EarthCurvatureCoef = 0.0
)

func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0
	lat1Rad := lat1 * math.Pi / 180.0
	lat2Rad := lat2 * math.Pi / 180.0
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return EarthRadius * c
}

func CalculateBearing(lat1, lon1, lat2, lon2 float64) float64 {
	dLon := (lon2 - lon1) * math.Pi / 180.0
	lat1Rad := lat1 * math.Pi / 180.0
	lat2Rad := lat2 * math.Pi / 180.0
	y := math.Sin(dLon) * math.Cos(lat2Rad)
	x := math.Cos(lat1Rad)*math.Sin(lat2Rad) - math.Sin(lat1Rad)*math.Cos(lat2Rad)*math.Cos(dLon)
	bearing := math.Atan2(y, x) * 180.0 / math.Pi
	return math.Mod(bearing+360.0, 360.0)
}

func EarthCurvatureDrop(distanceKm float64) float64 {
	return math.Pow(distanceKm, 2) * 1000 / (2 * EarthRadius * 1000) * 1000
}

func LineOfSightHeight(distanceKm, observerHeight float64) float64 {
	return observerHeight - EarthCurvatureDrop(distanceKm)
}

func CalculateVisibility(fromBeacon, toBeacon *models.Beacon, demPoints []models.DEMPoint) models.VisibilityAnalysis {
	distanceKm := HaversineDistance(fromBeacon.Lat, fromBeacon.Lon, toBeacon.Lat, toBeacon.Lon)
	bearing := CalculateBearing(fromBeacon.Lat, fromBeacon.Lon, toBeacon.Lat, toBeacon.Lon)
	earthDrop := EarthCurvatureDrop(distanceKm)

	fromEyeHeight := fromBeacon.Elevation + fromBeacon.Height
	toEyeHeight := toBeacon.Elevation + toBeacon.Height

	isVisible := false
	minClearance := 0.0
	maxTerrainElev := 0.0
	visibilityAngle := 0.0

	if len(demPoints) > 0 {
		maxTerrainElev = findMaxTerrainElevation(demPoints)
		lineOfSightAtMaxTerrain := calculateLineOfSightAtPoint(
			fromBeacon, toBeacon, maxTerrainElev, demPoints,
		)
		minClearance = lineOfSightAtMaxTerrain - maxTerrainElev
		isVisible = minClearance > 0
	} else {
		horizonDistanceFrom := math.Sqrt(2 * EarthRadius * 1000 * fromEyeHeight / 1000)
		horizonDistanceTo := math.Sqrt(2 * EarthRadius * 1000 * toEyeHeight / 1000)
		totalHorizonKm := (horizonDistanceFrom + horizonDistanceTo) / 1000
		isVisible = distanceKm <= totalHorizonKm
		minClearance = fromEyeHeight - earthDrop
		if !isVisible {
			minClearance = -math.Abs(minClearance - toEyeHeight)
		}
	}

	elevDiff := (toBeacon.Elevation + toBeacon.Height) - (fromBeacon.Elevation + fromBeacon.Height)
	visibilityAngle = math.Atan2(elevDiff, distanceKm*1000) * 180.0 / math.Pi

	return models.VisibilityAnalysis{
		FromBeaconID:        fromBeacon.ID,
		ToBeaconID:          toBeacon.ID,
		IsVisible:           isVisible,
		DistanceKm:          distanceKm,
		Bearing:             bearing,
		EarthCurvatureDrop:  earthDrop,
		MinClearance:        minClearance,
		MaxTerrainElevation: maxTerrainElev,
		VisibilityAngle:     visibilityAngle,
		CalculationMethod:   "dem_line_of_sight",
	}
}

func findMaxTerrainElevation(demPoints []models.DEMPoint) float64 {
	maxElev := -math.MaxFloat64
	for _, p := range demPoints {
		if p.Elevation > maxElev {
			maxElev = p.Elevation
		}
	}
	return maxElev
}

func calculateLineOfSightAtPoint(from, to *models.Beacon, terrainElev float64, demPoints []models.DEMPoint) float64 {
	fromHeight := from.Elevation + from.Height
	toHeight := to.Elevation + to.Height

	totalDist := HaversineDistance(from.Lat, from.Lon, to.Lat, to.Lon)

	minRatio := 1.0
	for _, p := range demPoints {
		distFrom := HaversineDistance(from.Lat, from.Lon, p.Lat, p.Lon)
		ratio := distFrom / totalDist
		if ratio > 0 && ratio < 1 {
			curvatureDrop := EarthCurvatureDrop(distFrom)
			lineHeight := fromHeight + ratio*(toHeight-fromHeight) - curvatureDrop
			clearance := lineHeight - p.Elevation
			if clearance/p.Elevation < minRatio {
				minRatio = clearance / p.Elevation
			}
		}
	}

	midDist := totalDist / 2
	midCurvature := EarthCurvatureDrop(midDist)
	midLineHeight := fromHeight + 0.5*(toHeight-fromHeight) - midCurvature

	return midLineHeight
}

func CalculateViewShedSector(beacon *models.Beacon, azimuthStart, azimuthEnd, maxDistanceKm float64) [][2]float64 {
	points := make([][2]float64, 0)
	numSteps := 36

	points = append(points, [2]float64{beacon.Lon, beacon.Lat})

	for i := 0; i <= numSteps; i++ {
		azimuth := azimuthStart + float64(i)*(azimuthEnd-azimuthStart)/float64(numSteps)
		lat, lon := destinationPoint(beacon.Lat, beacon.Lon, azimuth, maxDistanceKm)
		points = append(points, [2]float64{lon, lat})
	}

	points = append(points, [2]float64{beacon.Lon, beacon.Lat})

	return points
}

func destinationPoint(lat, lon, bearingDeg, distanceKm float64) (float64, float64) {
	latRad := lat * math.Pi / 180.0
	lonRad := lon * math.Pi / 180.0
	bearingRad := bearingDeg * math.Pi / 180.0
	angularDist := distanceKm / EarthRadius

	newLat := math.Asin(
		math.Sin(latRad)*math.Cos(angularDist) +
			math.Cos(latRad)*math.Sin(angularDist)*math.Cos(bearingRad),
	)
	newLon := lonRad + math.Atan2(
		math.Sin(bearingRad)*math.Sin(angularDist)*math.Cos(latRad),
		math.Cos(angularDist)-math.Sin(latRad)*math.Sin(newLat),
	)

	return newLat * 180.0 / math.Pi, newLon * 180.0 / math.Pi
}
