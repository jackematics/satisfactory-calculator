package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
)

func fractionToFloat(fraction string) float64 {
	split := strings.Split(fraction, "/")
	if len(split) == 1 {
		float, err := strconv.ParseFloat(split[0], 64)

		if err != nil {
			fmt.Println("Error parsing fraction")
			panic(err)
		}

		return float
	} else {
		lhs, err := strconv.ParseFloat(split[0], 64)
		rhs, err := strconv.ParseFloat(split[1], 64)

		if err != nil {
			fmt.Println("Error parsing fraction")
			panic(err)
		}

		return lhs / rhs
	}
}

type Ingredients map[string]string

type Building struct {
	Type string `json:"type"`
	Max string `json:"max"`
}

type ByProducts map[string]float64

func (byProducts ByProducts) Contains(item string) bool {
	for key, _ := range byProducts {
		if key == item {
			return true
		}
	}

	return false
}

type Recipe struct {
	Ingredients Ingredients `json:"ingredients"`
	Building Building `json:"building"`
	Alts []string `json:"alts,omitempty"`
	ByProducts map[string]string `json:"by-products,omitempty"`
}

type Recipes map[string]Recipe

type RawMaterials map[string]float64

type Row struct {
	Item string
	Quantity float64
	BldgType string
	BldgQuantity int
	Efficiency float64
}

type Table []Row

func (table Table) Find(item string) int {
	for i, row := range table {
		if row.Item == item {
			return i
		}
	}

	return -1
}


func (table Table) PrintTable() {
	fmt.Println("")
	fmt.Println("Recipe: " + strconv.FormatFloat(table[0].Quantity, 'f', 4, 64) + " " + table[0].Item)
	fmt.Println("")

	fmt.Printf("%-25s %-10s %-15s %-13s %-10s%%\n", "Item", "Quantity", "Bldg Type", "Bldg Quantity", "Efficiency")
	for _, row := range table {
		fmt.Printf(
			"%-25s %-10s %-15s %-13s %-10s%% \n", 
			row.Item, 
			strconv.FormatFloat(row.Quantity, 'f', 4, 64), 
			row.BldgType, 
			strconv.Itoa(row.BldgQuantity),
			strconv.FormatFloat(row.Efficiency, 'f', 4, 64),
		)
	}
}

func (recipes Recipes) buildRecipe(
	item string, 
	quantity float64, 
	alts []string, 
	table *Table, 
	rawMaterials *RawMaterials,
	byProducts *ByProducts,
	) (*Table, *RawMaterials, *ByProducts) {
	recipe, exists := recipes[item]
	tableIndex := table.Find(item)
	if tableIndex != -1 {
		quantity += (*table)[tableIndex].Quantity
	}

	if !exists {
		_, exists := (*rawMaterials)[item]
		if !exists {
			(*rawMaterials)[item] = quantity
		} else {
			(*rawMaterials)[item] += quantity
		}

		return table, rawMaterials, byProducts
	}

	for _, altA := range alts {
		for _, altB := range recipe.Alts {
			if altA == altB {
				item = altA
				recipe = recipes[altA]
			}
		}
	}

	buildingMax := fractionToFloat(recipe.Building.Max) 
	buildingQuantity := 1
	for quantity > buildingMax*float64(buildingQuantity) {
		buildingQuantity++
	}

	efficiency := (quantity / (buildingMax*float64(buildingQuantity))) * 100

	for key, value := range recipe.ByProducts {
		floatVal := fractionToFloat(value)
		if byProducts.Contains(key) {
			(*byProducts)[key] += floatVal * quantity
		} else {
			(*byProducts)[key] = floatVal * quantity
		}
	}

	if tableIndex == -1 {
		*table = append(*table, Row{
			Item: item, 
			Quantity: quantity,
			BldgType: recipe.Building.Type,
			BldgQuantity: buildingQuantity,
			Efficiency: efficiency,
		})
	} else {
		(*table)[tableIndex] = Row{
			Item: item,
			Quantity: quantity,
			BldgType: recipe.Building.Type,
			BldgQuantity: buildingQuantity,
			Efficiency: efficiency,
		}
	}

	for key, value := range recipe.Ingredients {
		table, rawMaterials, byProducts = recipes.buildRecipe(key, quantity * fractionToFloat(value), alts, table, rawMaterials, byProducts)
	}

	return table, rawMaterials, byProducts
} 

func getRecipes() (Recipes, error) {
	file, err := os.Open("recipes.json")
	if err != nil {
		fmt.Println("Error retrieving recipes")
		return nil ,err
	}
	defer file.Close()

	var recipes Recipes
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&recipes)
	if err != nil {
		fmt.Println("Error retrieving recipes")
		return nil, err
	}

	return recipes, nil
}

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name: "alts",
				Aliases: []string{"a"},
				Usage: "Swap to alternate recipes",
			},
			&cli.Float64Flag{
				Name: "quantity",
				Aliases: []string{"q"},
				Usage: "Quantity of item to produce",
				Value: 1.0,
			},
		},
		Action: func(cCtx *cli.Context) error {
			item := cCtx.Args().Get(0)

			if item == "" {
				log.Fatal("invalid item")
			}

			alts := cCtx.StringSlice("alts")

			if alts == nil {
				alts = []string{}
			}
			
			recipes, err := getRecipes()

			if err != nil {
				log.Fatal(err)
			}

			_, itemExists := recipes[item]
			if !itemExists {
				log.Fatal("item does not exist")
			}

			quantity := cCtx.Float64("quantity")
			
			table, rawMaterials, byProducts := recipes.buildRecipe(
				item, 
				quantity, 
				alts,
				&Table{},
				&RawMaterials{},
				&ByProducts{},
			)

			table.PrintTable()

			fmt.Println("")
			fmt.Println("Raw materials: ")
			for key, value := range *rawMaterials {
				fmt.Printf("%-25s %-10s\n", key, strconv.FormatFloat(value, 'f', 4, 64))
			}

			fmt.Println("")
			fmt.Println("By products: ")
			for key, value := range *byProducts {
				fmt.Printf("%-25s %-10s\n", key, strconv.FormatFloat(value, 'f', 4, 64))
			}
			fmt.Println("")

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}