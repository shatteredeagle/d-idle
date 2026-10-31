// Package d2map reads the Diablo II world map (levels, rooms, collision, preset units) out of
// game memory through a mem.Source and merges it into walkability grids that navigation can use.
//
// Coordinates: everything exported here is in absolute game subtiles (the unit of
// Path.xPos/yPos); one level tile is 5x5 subtiles. Rects are half-open: [X, X+W) x [Y, Y+H).
//
// An external reader cannot ask the client to load rooms, so a Grid only contains the rooms the
// client has ever had active while a Tracker was watching. Everything else is explicitly unknown.
package d2map
