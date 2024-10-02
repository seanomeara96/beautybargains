package main

import "math/rand"

// get all
func getPersonas(limit, offset int) []Persona {
	personas := []Persona{
		{1, "Michaela Gormley", "", "https://replicate.delivery/yhqm/uZSufO1ULetrN00FXde5s7mjJJB5i0yh3Drfms5HeBb0ReS1E/out-0.webp"},
		{2, "Caroline Mullen ", "", "https://replicate.delivery/yhqm/kqyPgcRDPFZgAxqY5aQqgLtscg6x2zCal815fwTDDQZ15lqJA/out-0.webp"},
		{3, "Susan Fagan", "", "https://replicate.delivery/yhqm/sxYBygzKehzXHiJk2zdLF4sK3LeZ0jx6ss9qgmbJKcreoXqmA/out-0.webp"},
		{4, "Clare O'Shea", "", "https://replicate.delivery/yhqm/JqI1VW2t0f3lfkO4T7WfVciDf9BdsuzYcWljhcrFfGjfR9S1E/out-0.webp"},
		{5, "Aisling O'Reilly", "", "https://replicate.delivery/yhqm/nXCbkLusEIItOJf0UY6JCiXs1oaxrpKLXwV8dkhKMe791LVTA/out-0.webp"},
		{6, "Stacey Dowling", "", "https://replicate.delivery/yhqm/gMBf17FLmVQnGS9te4MWQJWlE6PdhjZd5Fhl2ycIZj1ftXqmA/out-0.webp"},
		{7, "Danielle Duffy", "", "https://replicate.delivery/yhqm/rvXUljqA3QqxCtGa64qnmJyn5jh571lvA8Dixinh6LK39S1E/out-0.webp"},
	}

	lenPersonas := len(personas)

	if limit == 0 || limit > lenPersonas {
		limit = lenPersonas
	}

	if offset > len(personas) {
		offset = 0
	}

	res := []Persona{}
	for i := offset; i < limit; i++ {
		res = append(res, personas[i])
	}

	return res
}

// get one random persona
func getRandomPersona() Persona {
	personas := getPersonas(0, 0)
	randomInt := rand.Intn(len(personas))
	return personas[randomInt]
}
