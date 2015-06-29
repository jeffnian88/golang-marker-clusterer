package mylib

import (
	"fmt"
	"math"
        "labix.org/v2/mgo"
        "labix.org/v2/mgo/bson"
)
// google api v3 
// start
type Gps_LatLng struct {
	Lng float64
	Lat float64
}
func (g *Gps_LatLng) lat() float64 {return g.Lat}
func (g *Gps_LatLng) lng() float64 {return g.Lng}
func (g *Gps_LatLng) print() {fmt.Printf("(%f, %f)", g.Lat, g.Lng)}

// Mecator Projection api
type PointF struct {
	X float64
	Y float64
}
type MercatorProjection struct {
	PixelTileSize  int
	DegreesToRadiansRatio float64
	RadiansToDegreesRatio float64

	pixelGlobeSize float64
	XPixelsToDegreesRatio float64
	YPixelsToRadiansRatio float64
	halfPixelGlobeSize float64
	PixelGlobeCenter PointF
}
func NewMP(level int) *MercatorProjection {
	pMP := &MercatorProjection{}

	pMP.PixelTileSize = 256
	pMP.DegreesToRadiansRatio = 180 / math.Pi
	pMP.RadiansToDegreesRatio = math.Pi / 180
	// 256*2^16 = 2^8*2^16 = 2^24
	pMP.pixelGlobeSize = float64(pMP.PixelTileSize) * math.Pow(2.0, float64(level))
	pMP.XPixelsToDegreesRatio = pMP.pixelGlobeSize / 360
	pMP.YPixelsToRadiansRatio = pMP.pixelGlobeSize / (2*math.Pi)
	pMP.halfPixelGlobeSize = pMP.pixelGlobeSize / 2
	pMP.PixelGlobeCenter = PointF{pMP.halfPixelGlobeSize, pMP.halfPixelGlobeSize}
	return pMP
}

func (m *MercatorProjection) FromCoordinatesToPixel(coordinates PointF) PointF {
	x := (m.PixelGlobeCenter.X + (coordinates.X * m.XPixelsToDegreesRatio))
    //f := math.Min(math.Max(math.Sin(coordinates.Y * m.RadiansToDegreesRatio), -0.9999), 0.9999)
    f := math.Sin(coordinates.Y * m.RadiansToDegreesRatio)
    y := (m.PixelGlobeCenter.Y + (0.5) * math.Log((1 + f) / (1 - f)) * -m.YPixelsToRadiansRatio)
    return PointF{x, y}
}

func (m *MercatorProjection) FromPixelToCoordinates(pixel PointF) PointF {
    longitude := (pixel.X - m.PixelGlobeCenter.X) / m.XPixelsToDegreesRatio
    latitude := (2 * math.Atan(math.Exp((pixel.Y - m.PixelGlobeCenter.Y) / -m.YPixelsToRadiansRatio)) - math.Pi / 2) * m.DegreesToRadiansRatio
	if longitude > 180 {
		longitude = longitude - 360
	}
	if longitude < -180 {
		longitude = longitude + 360
	}
    return PointF{latitude,longitude}
}

func distanceBetweenPoints(p1 *Gps_LatLng, p2 *Gps_LatLng) float64 {
	if (p1 == nil || p2 == nil) { return 0 }

	var R = float64(6371.0)	 // Radius of the Earth in km
	var dLat = (p2.lat() - p1.lat()) * math.Pi / 180
	var dLon = (p2.lng() - p1.lng()) * math.Pi / 180
	var a = math.Sin(dLat / 2) * math.Sin(dLat / 2) +
		math.Cos(p1.lat() * math.Pi / 180) * math.Cos(p2.lat() * math.Pi / 180) *
		math.Sin(dLon / 2) * math.Sin(dLon / 2)
	var c = 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1 - a))
	var d = R * c
	return d
}
type Gps_LatLngBounds struct {
	NorthEast_ Gps_LatLng
	SouthWest_ Gps_LatLng
	PMP *MercatorProjection
}
func (g *Gps_LatLngBounds) getNorthEast() Gps_LatLng { return g.NorthEast_}
func (g *Gps_LatLngBounds) getSouthWest() Gps_LatLng { return g.SouthWest_}
func (g *Gps_LatLngBounds) Extendwh(width int, height int, level int) *Gps_LatLngBounds {
        // sanity-check
        //if level > 0 {log.Fatal("extend: ")}

        // initialize MP
        if g.PMP == nil {
                g.PMP = NewMP(level)
        }
        //
        var point PointF

        // NE
        ne := g.getNorthEast()
        point = g.PMP.FromCoordinatesToPixel(PointF{ne.lng(), ne.lat()})
        point = g.PMP.FromPixelToCoordinates(PointF{point.X+float64(width), point.Y-float64(height)})
	ne_ := Gps_LatLng{Lat:point.X, Lng:point.Y}
        // SW
        sw := g.getSouthWest()
        point = g.PMP.FromCoordinatesToPixel(PointF{sw.lng(), sw.lat()})
        point = g.PMP.FromPixelToCoordinates(PointF{point.X-float64(width), point.Y+float64(height)})
	sw_ := Gps_LatLng{Lat:point.X, Lng:point.Y}

	return &Gps_LatLngBounds{Gps_LatLng{Lat:ne_.lat(), Lng:ne_.lng()}, Gps_LatLng{Lat:sw_.lat(), Lng:sw_.lng()}, nil}
}
func (g *Gps_LatLngBounds) extend(gridpixel int, level int) *Gps_LatLngBounds {

	// sanity-check 
	//if level > 0 {log.Fatal("extend: ")}

	// initialize MP
	if g.PMP == nil {
		g.PMP = NewMP(level)
	}

	//
	var point PointF

	// NE
	ne := g.getNorthEast()
	point = g.PMP.FromCoordinatesToPixel(PointF{ne.lng(), ne.lat()})
	point = g.PMP.FromPixelToCoordinates(PointF{point.X+float64(gridpixel), point.Y-float64(gridpixel)})
	ne_ := Gps_LatLng{Lat:point.X, Lng:point.Y}
	// SW
	sw := g.getSouthWest()
	point = g.PMP.FromCoordinatesToPixel(PointF{sw.lng(), sw.lat()})
	point = g.PMP.FromPixelToCoordinates(PointF{point.X-float64(gridpixel), point.Y+float64(gridpixel)})
	sw_ := Gps_LatLng{Lat:point.X, Lng:point.Y}

	return &Gps_LatLngBounds{Gps_LatLng{Lat:ne_.lat(), Lng:ne_.lng()}, Gps_LatLng{Lat:sw_.lat(), Lng:sw_.lng()}, nil}
}

func (g *Gps_LatLngBounds) contains(p *Gps_LatLng) bool {
	ne := g.getNorthEast()
	sw := g.getSouthWest()

	// lat sanity-check
	if sw.lat() > ne.lat() {
		//log.Fatal("sw.lat() > ne.lat()")
		return false
	}

	// check lat
	if sw.lat() <= p.lat() && p.lat() <= ne.lat() {
		// need to check lng
	} else {
		// p.lat is not in sw.lat()~ne.lat()
		// must not contain point
		return false
	}

	// check lng
	// point in edge of bounds
	if p.lng() == sw.lng() || p.lng() == ne.lng(){
		return true
	}
	if sw.lng() <= ne.lng() {
		if sw.lng() <= p.lng() && p.lng() <= ne.lng() {
			return true
		} else {
			return false
		}
	}
	if sw.lng() > ne.lng() {
		otherBounds := Gps_LatLngBounds{Gps_LatLng{Lat:ne.lat(), Lng:sw.lng()}, Gps_LatLng{Lat:sw.lat(), Lng:ne.lng()}, nil}
		return !otherBounds.contains(p)
	}
	return false
}
// google api end
var count int
// markercluster lib start
// use google api
type Cluster struct {
	Id int
	Gps_ *Gps_LatLng
	Bounds *Gps_LatLngBounds
	Weight int
}

func (c *Cluster) calculateBounds(page *Page, gridsize int) bool {
	bounds := Gps_LatLngBounds{Gps_LatLng{Lat:c.Gps_.lat(), Lng:c.Gps_.lng()}, Gps_LatLng{Lat:c.Gps_.lat(), Lng:c.Gps_.lng()}, nil}
	c.Bounds = bounds.extend(gridsize, page.Level)
	return true
}

func (c *Cluster) addMarker(marker *Marker, page *Page, gridsize int, clustersize int) bool {
/*
  if (this.isMarkerAlreadyAdded(marker)) {
    return false;
  }
*/
  // innsert marker to cluster first time
  if c.Gps_ == nil {
	  c.Gps_ = &Gps_LatLng{Lat:marker.Gps_.lat(), Lng:marker.Gps_.lng()}
	c.Weight = 1
  } else {
 //   if (this.averageCenter_) {
      l := float64(c.Weight + 1)
      lat := (c.Gps_.lat() * (l-1) + marker.Gps_.lat()) / l
      lng := (c.Gps_.lng() * (l-1) + marker.Gps_.lng()) / l
      c.Gps_ = &Gps_LatLng{Lat:lat, Lng:lng}
      c.Weight = c.Weight + 1
 //     this.calculateBounds_();
 //   }
  }
  // either empty or non-empty cluster, we need to recalculate the bounds of cluster
   c.calculateBounds(page, gridsize)
//   c.printCluster()

/*
  marker.isAdded = true;
  this.markers_.push(marker);
  marker.setMap(null);
*/
//    c.Weight = clusterToAddTo.weight + 1
  return true
}

func (c *Cluster) printCluster() bool {
	ne := c.Bounds.getNorthEast()
	sw := c.Bounds.getSouthWest()
	fmt.Printf("ID(%3d) Weight:%3d \tCluster center (%f,%f)\tbound(%f,%f,%f,%f)\n", c.Id, c.Weight, c.Gps_.lat(), c.Gps_.lng(), ne.lat(), ne.lng(), sw.lat(), sw.lng())
	return true
}

type Marker struct {
	Gps_ Gps_LatLng
	Weight int
	Id int
}

// following element may be in database or in memory, or both
type Page struct {
	Level int
	Clustersize	 int
	Gridsize	 int
	SliceClusters []*Cluster
	Sclen	int
	Tolcls int
/*
	pparent *Page
	ptl_next *Page
	ptr_next *Page
	pdl_next *Page
	pdr_next *Page
*/
	Usedb bool
	Maxsc int
	Col *mgo.Collection

	Dirty_ bool
}
type DBWrapperEntry struct {
	Id int
	Loc tLoc
	Weight int
}

func (p *Page) getNearCluster(gps *Gps_LatLng) *Cluster {
        one := Cluster{}
	err := p.Col.Find(bson.M{"gps_": bson.M{"$near" : Gps_LatLng{Lng:gps.lng(), Lat:gps.lat()}}}).One(&one)
        if err != nil {return nil}
        //one.printCluster()
	return &one
}
func (p *Page) flushCluster() bool {
	for _, c := range p.SliceClusters {
		err := p.Col.Insert(c)
		if err != nil {
			fmt.Printf("insert err ")
		}
	}
	p.SliceClusters = nil
	p.Sclen = 0
	return true
}
func (p *Page) updateCluster(c *Cluster) bool {
	/*
	err := p.Col.Remove(bson.M{"id": c.Id})
	if err != nil {
		fmt.Println(err)
	}
	err = p.Col.Insert(c)
	if err != nil {
		fmt.Println(err)
	}
	*/
	err := p.Col.Update(bson.M{"id": c.Id}, bson.M{"$inc": bson.M{"weight":1}})
	if err != nil {
		fmt.Println(err)
	}
	return true
}
func (p *Page) PrintPage() bool {
	fmt.Printf("Page Level: %d\n", p.Level)
	// print cluster
	for i:= 0; i<len(p.SliceClusters); i++ {
		p.SliceClusters[i].printCluster()
	}
	return true
}

func (p *Page) SearchCluster(bounds *Gps_LatLngBounds) *[]*Cluster {

	if p.Usedb == true {
        	var many []*Cluster
		err := p.Col.Find(bson.M{"gps_": bson.M{"$near" : Gps_LatLng{Lng:0, Lat:0}}}).All(&many)
        	if err != nil {return nil}
	        //one.printCluster()
		return &many
	}
	var sliceCluster = make([]*Cluster, 0)
	for _, c := range p.SliceClusters {
		if true == bounds.contains(c.Gps_) {
			sliceCluster = append(sliceCluster, c)
		}
	}
	return &sliceCluster
}

// BottomUp methodology

func (p *Page) InsertMarkers(markers []*Marker) bool {
	if p.Usedb == true {
		index := mgo.Index{Key: []string{"$2d:gps_"}, Bits: 26}
		err := p.Col.EnsureIndex(index)
		if err != nil {
			fmt.Println(err)
		}
	}
	for _, m := range markers {
		p.createPageCluster(m)
	}
	if p.Usedb == true {
		p.flushCluster()
	}
	return true
}

func (p *Page) createPageCluster(marker *Marker) bool {

  var distance float64
  var clusterToAddTo *Cluster
  distance = 40000		// Some large number
  clusterToAddTo = nil

  // sanity-check
  // add code here

  //var pos = marker.getPosition();
  var pcluster *Cluster
  var findindb bool
  var dbnearest *Cluster

  findindb = false
  // find which Cluster is closet to marker
  for i := 0 ; i<len(p.SliceClusters) ; i++  {
  	pcluster = p.SliceClusters[i]
//    center := pcluster.Gps_
//    if center != nil { //?? need to check
      d := distanceBetweenPoints(pcluster.Gps_, &marker.Gps_)
      if d < distance {
//	  	fmt.Printf("distance:%f ", d)
        distance = d
        clusterToAddTo = pcluster
      }
//    }
  }
  if p.Usedb == true && (p.Tolcls-p.Sclen)>0{
	dbnearest = p.getNearCluster(&marker.Gps_)
	if dbnearest != nil {
		dd :=  distanceBetweenPoints(dbnearest.Gps_, &marker.Gps_)
		if dd < distance {
		        distance = dd
		        clusterToAddTo = dbnearest
			findindb = true
		}
	}
  }
/*
  if  nil != clusterToAddTo {
   fmt.Printf("bounds:%t", clusterToAddTo.Bounds.contains(&marker.Gps_))
  }
*/
  if nil != clusterToAddTo && true == clusterToAddTo.Bounds.contains(&marker.Gps_) {
	clusterToAddTo.addMarker(marker, p, p.Gridsize, p.Clustersize)
  	if findindb == true {p.updateCluster(clusterToAddTo)}
//	fmt.Printf("ToAddTo\n")
  } else {
	var cluster = &Cluster{Gps_:nil, Id:p.Tolcls}
	p.Tolcls = p.Tolcls+ 1
	cluster.addMarker(marker, p, p.Gridsize, p.Clustersize);
	p.SliceClusters = append(p.SliceClusters, cluster)
	p.Sclen = p.Sclen + 1
//    this.clusters_.push(cluster)
  }

  if p.Usedb == true && p.Sclen >= p.Maxsc {
	p.flushCluster()
  }

  return true
}

type WebWrapperEntry struct {
	Lat float64
	Lng float64
	Weight int
	NELat float64
	NELng float64
	SWLat float64
	SWLng float64
}
func GetWebWrapperEntry(sliceCluster *[]*Cluster, zoom int) []WebWrapperEntry{
	var entries []WebWrapperEntry
	fmt.Println("########################")
	for _, c := range *sliceCluster {
	ne := c.Bounds.getNorthEast()
	sw := c.Bounds.getSouthWest()
		//if zoom < 3 && c.Weight < 5 {continue}
		entries = append(entries, WebWrapperEntry{c.Gps_.lat(), c.Gps_.lng(), c.Weight, ne.lat(), ne.lng(), sw.lat(), sw.lng()})
		c.printCluster()
	}
	return entries
}

type tLoc struct {
	Lng float64
	Lat float64
}
type Place struct {
        Num int
        Loc tLoc
}
func (p *Place) Print() {
	fmt.Printf("(%d) LngLat(%f, %f)", p.Num, p.Loc.Lng, p.Loc.Lat)
}
func findallradius(sm []*Marker, dist float64) {
	center := &Gps_LatLng{0, 0}
	for _, m := range sm {
		d := distanceBetweenPoints(center, &m.Gps_)
		if d <= dist {
			fmt.Printf("(%3d) LngLat(%f, %f)\n", m.Id, m.Gps_.lng(), m.Gps_.lat())
		}
	}
}
