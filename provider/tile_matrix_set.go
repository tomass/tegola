package provider

import (
	"fmt"
	"math"

	"github.com/go-spatial/geom"
	"github.com/go-spatial/tegola"
)

// TileMatrixSet is a tile grid in a CRS other than WebMercator, defined the way
// WMTS / OGC TMS grids are: a top-left origin, a tile size in pixels and a
// resolution (CRS units per pixel) for every zoom level. Extent bounds the
// number of tiles at each zoom.
type TileMatrixSet struct {
	ID          string     `toml:"id"`
	SRID        uint64     `toml:"srid"`
	TileSize    uint       `toml:"tile_size"`
	Origin      [2]float64 `toml:"origin"`
	Extent      [4]float64 `toml:"extent"`
	Resolutions []float64  `toml:"resolutions"`
}

// tileMatrixSets holds the configured grids keyed by SRID, since a Tile only
// carries the SRID of its map. This limits a config to one grid per CRS.
var tileMatrixSets = map[uint64]*TileMatrixSet{}

// SetTileMatrixSets validates the configured grids and replaces the registered ones.
func SetTileMatrixSets(sets []TileMatrixSet) error {
	reg := make(map[uint64]*TileMatrixSet, len(sets))
	for i := range sets {
		t := &sets[i]
		switch {
		case t.ID == "":
			return fmt.Errorf("tile matrix set #%d: id is required", i)
		case t.SRID == 0 || t.SRID == tegola.WebMercator:
			return fmt.Errorf("tile matrix set %v: srid is required and can not be %v", t.ID, tegola.WebMercator)
		case t.TileSize == 0 || len(t.Resolutions) == 0:
			return fmt.Errorf("tile matrix set %v: tile_size and resolutions are required", t.ID)
		case t.Origin[0] >= t.Extent[2] || t.Origin[1] <= t.Extent[1]:
			return fmt.Errorf("tile matrix set %v: origin must be the top-left corner of extent", t.ID)
		case reg[t.SRID] != nil:
			return fmt.Errorf("tile matrix set %v: only one tile matrix set per srid (%v) is supported", t.ID, t.SRID)
		}
		reg[t.SRID] = t
	}
	tileMatrixSets = reg
	return nil
}

// TileMatrixSetForSRID returns the grid registered for srid, or nil for WebMercator.
func TileMatrixSetForSRID(srid uint64) *TileMatrixSet { return tileMatrixSets[srid] }

// TileMatrixSetByID returns the grid registered under id, or nil.
func TileMatrixSetByID(id string) *TileMatrixSet {
	for _, t := range tileMatrixSets {
		if t.ID == id {
			return t
		}
	}
	return nil
}

// tileSpan is the width (and height) of a tile at zoom z in CRS units.
func (t *TileMatrixSet) tileSpan(z uint) (float64, error) {
	if int(z) >= len(t.Resolutions) {
		return 0, fmt.Errorf("tile matrix set %v has no zoom %v", t.ID, z)
	}
	return t.Resolutions[z] * float64(t.TileSize), nil
}

// MatrixSize returns the number of tile columns and rows at zoom z.
func (t *TileMatrixSet) MatrixSize(z uint) (cols, rows uint, err error) {
	span, err := t.tileSpan(z)
	if err != nil {
		return 0, 0, err
	}
	return uint(math.Ceil((t.Extent[2] - t.Origin[0]) / span)), uint(math.Ceil((t.Origin[1] - t.Extent[1]) / span)), nil
}

// TileExtent returns the extent of tile z/x/y in the grid's CRS.
func (t *TileMatrixSet) TileExtent(z, x, y uint) (*geom.Extent, error) {
	span, err := t.tileSpan(z)
	if err != nil {
		return nil, err
	}
	minX, maxY := t.Origin[0]+float64(x)*span, t.Origin[1]-float64(y)*span
	return geom.NewExtent([2]float64{minX, maxY - span}, [2]float64{minX + span, maxY}), nil
}
