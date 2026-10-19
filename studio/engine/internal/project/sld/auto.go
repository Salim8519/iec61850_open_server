package sld

import (
	"sort"

	"studio-engine/internal/scl"
)

// sld_auto_layout: deterministic grid placement for equipment without drawing
// coordinates. Units are grid cells; voltage levels are rows, bays columns, equipment is
// stacked by its distance from a busbar.
const (
	slot     = 2.0
	step     = 2.5
	busStep  = 1.5
	bayGap   = 1.0
	levelGap = 2.0
)

type point = [2]float64

// busNodes is bus_nodes: nodes in a bay without drawn equipment, or spanning bays.
func busNodes(equipment []*scl.Equipment, members *ordered, nodes map[string]*scl.ConnectivityNode) []string {
	bays := map[[2]int]bool{}
	byPath := map[string]*scl.Equipment{}
	for _, item := range equipment {
		bays[[2]int{item.Level, item.Bay}] = true
		byPath[item.Path] = item
	}
	var result []string
	for _, path := range members.keys {
		node := nodes[path]
		spanned := map[[2]int]bool{}
		for _, member := range members.values[path] {
			spanned[[2]int{byPath[member].Level, byPath[member].Bay}] = true
		}
		if !bays[[2]int{node.Level, node.Bay}] || len(spanned) > 1 {
			result = append(result, path)
		}
	}
	return result
}

// orderOf is _order: the last index of each path in equipment.
func orderOf(equipment []*scl.Equipment) map[string]int {
	index := map[string]int{}
	for number, item := range equipment {
		index[item.Path] = number
	}
	return index
}

func sortedBy(paths []string, order map[string]int) []string {
	result := append([]string(nil), paths...)
	sort.SliceStable(result, func(i, j int) bool { return order[result[i]] < order[result[j]] })
	return result
}

// depths is _depths: breadth-first distance from the busbars.
func depths(equipment []*scl.Equipment, members *ordered, buses map[string]bool) map[string]int {
	neighbours := map[string][]string{}
	for _, path := range members.keys {
		for _, item := range members.values[path] {
			neighbours[item] = append(neighbours[item], path)
		}
	}
	order := orderOf(equipment)
	depth := map[string]int{}
	seen := map[string]bool{}
	var frontier []string
	for _, path := range members.keys {
		if buses[path] {
			for _, member := range members.values[path] {
				if !seen[member] {
					seen[member] = true
					frontier = append(frontier, member)
				}
			}
		}
	}
	queue := sortedBy(frontier, order)
	for _, item := range queue {
		depth[item] = 0
	}
	all := make([]string, len(equipment))
	for index, item := range equipment {
		all[index] = item.Path
	}
	for _, start := range sortedBy(all, order) {
		if _, known := depth[start]; !known {
			depth[start] = 0
			queue = append(queue, start)
		}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			for _, node := range neighbours[current] {
				if buses[node] {
					continue
				}
				for _, other := range sortedBy(members.values[node], order) {
					if _, known := depth[other]; !known {
						depth[other] = depth[current] + 1
						queue = append(queue, other)
					}
				}
			}
		}
	}
	return depth
}

// autoPositions is auto_positions: equipment positions and busbar heights in grid cells.
func autoPositions(equipment []*scl.Equipment, members *ordered,
	nodes map[string]*scl.ConnectivityNode) (map[string]point, map[string]float64) {
	buses := busNodes(equipment, members, nodes)
	busSet := map[string]bool{}
	for _, path := range buses {
		busSet[path] = true
	}
	depth := depths(equipment, members, busSet)
	positions, busY := map[string]point{}, map[string]float64{}
	top := 0.0
	levelSet := map[int]bool{}
	for _, item := range equipment {
		levelSet[item.Level] = true
	}
	for _, path := range buses {
		levelSet[nodes[path].Level] = true
	}
	for _, level := range sortedInts(levelSet) {
		var levelBuses []string
		for _, path := range buses {
			if nodes[path].Level == level {
				levelBuses = append(levelBuses, path)
			}
		}
		for row, path := range levelBuses {
			busY[path] = top + float64(float64(row)*busStep) + 0.5
		}
		first := top + float64(float64(len(levelBuses))*busStep) + 1.0
		left, deepest := 0.0, 0
		baySet := map[int]bool{}
		for _, item := range equipment {
			if item.Level == level {
				baySet[item.Bay] = true
			}
		}
		for _, bay := range sortedInts(baySet) {
			var groupKeys []int
			groups := map[int][]string{}
			for _, item := range equipment {
				if item.Level == level && item.Bay == bay {
					key := depth[item.Path]
					if _, seen := groups[key]; !seen {
						groupKeys = append(groupKeys, key)
					}
					groups[key] = append(groups[key], item.Path)
				}
			}
			widest := 0
			for _, key := range groupKeys {
				for index, path := range groups[key] {
					positions[path] = point{left + float64(float64(index)*slot), first + float64(float64(key)*step)}
				}
				deepest = max(deepest, key)
				widest = max(widest, len(groups[key]))
			}
			left += float64(float64(widest)*slot) + bayGap
		}
		top = first + float64(float64(deepest+1)*step) + levelGap
	}
	return positions, busY
}

func sortedInts(set map[int]bool) []int {
	result := make([]int, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Ints(result)
	return result
}
