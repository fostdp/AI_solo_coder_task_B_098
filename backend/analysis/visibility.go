package analysis

import (
	"beacon-system/models"
	"math"
)

const (
	EarthRadius = 6371.0

	EffectiveEarthFactorK = 4.0 / 3.0

	RefractionGradient = -40e-6
)

func EffectiveEarthRadius() float64 {
	return EarthRadius * EffectiveEarthFactorK
}

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

func ITURRefractedCurvatureDrop(distanceKm float64) float64 {
	ae := EffectiveEarthRadius() * 1000
	dMeters := distanceKm * 1000
	return (dMeters * dMeters) / (2 * ae)
}

func ITURRefractedHorizonDistance(heightMeters float64) float64 {
	ae := EffectiveEarthRadius() * 1000
	return math.Sqrt(2*ae*heightMeters) / 1000
}

func ITURRefractedLineOfSightHeight(distanceKm, observerHeightM float64) float64 {
	ae := EffectiveEarthRadius() * 1000
	dMeters := distanceKm * 1000
	return observerHeightM - (dMeters*dMeters)/(2*ae)
}

func ITURProfileHeight(distanceKm, observerHeightM, targetHeightM, totalDistKm float64) float64 {
	if totalDistKm <= 0 {
		return observerHeightM
	}
	ratio := distanceKm / totalDistKm
	linearHeight := observerHeightM + ratio*(targetHeightM-observerHeightM)
	ae := EffectiveEarthRadius() * 1000
	dMeters := distanceKm * 1000
	curvatureDrop := (dMeters * dMeters) / (2 * ae)
	return linearHeight - curvatureDrop
}

func LineOfSightHeight(distanceKm, observerHeight float64) float64 {
	return ITURRefractedLineOfSightHeight(distanceKm, observerHeight)
}

func CalculateVisibility(fromBeacon, toBeacon *models.Beacon, demPoints []models.DEMPoint) models.VisibilityAnalysis {
	distanceKm := HaversineDistance(fromBeacon.Lat, fromBeacon.Lon, toBeacon.Lat, toBeacon.Lon)
	bearing := CalculateBearing(fromBeacon.Lat, fromBeacon.Lon, toBeacon.Lat, toBeacon.Lon)

	refractedDrop := ITURRefractedCurvatureDrop(distanceKm)

	fromEyeHeight := fromBeacon.Elevation + fromBeacon.Height
	toEyeHeight := toBeacon.Elevation + toBeacon.Height

	isVisible := false
	minClearance := 0.0
	maxTerrainElev := 0.0
	visibilityAngle := 0.0

	if len(demPoints) > 0 {
		minClearance, maxTerrainElev = checkDEMProfileWithRefraction(
			fromBeacon, toBeacon, demPoints, distanceKm,
		)
		isVisible = minClearance > 0
	} else {
		horizonFromKm := ITURRefractedHorizonDistance(fromEyeHeight)
		horizonToKm := ITURRefractedHorizonDistance(toEyeHeight)
		totalHorizonKm := horizonFromKm + horizonToKm
		isVisible = distanceKm <= totalHorizonKm

		if isVisible {
			midProfileHeight := ITURProfileHeight(distanceKm/2, fromEyeHeight, toEyeHeight, distanceKm)
			minClearance = midProfileHeight
		} else {
			midProfileHeight := ITURProfileHeight(distanceKm/2, fromEyeHeight, toEyeHeight, distanceKm)
			minClearance = midProfileHeight
		}

		if !isVisible {
			minClearance = -math.Abs(minClearance)
		}
	}

	elevDiff := toEyeHeight - fromEyeHeight
	visibilityAngle = math.Atan2(elevDiff, distanceKm*1000) * 180.0 / math.Pi

	return models.VisibilityAnalysis{
		FromBeaconID:        fromBeacon.ID,
		ToBeaconID:          toBeacon.ID,
		IsVisible:           isVisible,
		DistanceKm:          distanceKm,
		Bearing:             bearing,
		EarthCurvatureDrop:  refractedDrop,
		MinClearance:        minClearance,
		MaxTerrainElevation: maxTerrainElev,
		VisibilityAngle:     visibilityAngle,
		CalculationMethod:   "itu_r_refracted_los",
	}
}

func checkDEMProfileWithRefraction(from, to *models.Beacon, demPoints []models.DEMPoint, totalDistKm float64) (float64, float64) {
	fromEyeHeight := from.Elevation + from.Height
	toEyeHeight := to.Elevation + to.Height

	maxTerrainElev := -math.MaxFloat64
	minClearance := math.MaxFloat64

	for _, p := range demPoints {
		if p.Elevation > maxTerrainElev {
			maxTerrainElev = p.Elevation
		}

		distFrom := HaversineDistance(from.Lat, from.Lon, p.Lat, p.Lon)
		ratio := distFrom / totalDistKm
		if ratio > 0.01 && ratio < 0.99 {
			profileHeight := ITURProfileHeight(distFrom, fromEyeHeight, toEyeHeight, totalDistKm)
			clearance := profileHeight - p.Elevation
			if clearance < minClearance {
				minClearance = clearance
			}
		}
	}

	if minClearance == math.MaxFloat64 {
		midProfileHeight := ITURProfileHeight(totalDistKm/2, fromEyeHeight, toEyeHeight, totalDistKm)
		minClearance = midProfileHeight - maxTerrainElev
	}

	return minClearance, maxTerrainElev
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
