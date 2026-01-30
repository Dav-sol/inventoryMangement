package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Item struct {
	ID         int
	Name       string
	SKU        string
	Quantity   int
	Price      float64
	Supplier   string
	Category   string
	ReorderAt  int
	LeadTime   int
	DateAdded  time.Time
	IsLowStock bool
}

type Store struct {
	items  map[int]*Item
	nextID int
}

type Stats struct {
	Total      int
	LowStock   int
	TotalValue float64
	Categories int
}

type CategoryData struct {
	Name        string
	Count       int
	AvgLeadTime float64
}

var store = &Store{items: make(map[int]*Item), nextID: 1}

const pageTemplate = `<!DOCTYPE html>
<html><head><title>Inventory System</title>
<style>
body{font-family:Arial;margin:20px;background:#f5f5f5;}
.container{max-width:1200px;margin:0 auto;background:white;padding:20px;border-radius:8px;}
.tabs{display:flex;margin-bottom:20px;}
.tab{padding:10px 20px;background:#ddd;cursor:pointer;border:1px solid #ccc;margin-right:5px;text-decoration:none;color:#333;}
.tab.active{background:#007bff;color:white;}
.tab-content{display:none;}
.tab-content.active{display:block;}
table{border-collapse:collapse;width:100%;margin:10px 0;}
th,td{border:1px solid #ddd;padding:8px;text-align:left;}
th{background:#f2f2f2;}
input,select{padding:5px;margin:2px;}
button{padding:5px 10px;margin:2px;cursor:pointer;}
.btn-primary{background:#007bff;color:white;border:1px solid #007bff;}
.low{background:#ffcccc;}
.form{background:#f9f9f9;padding:15px;margin:10px 0;border:1px solid #ccc;}
.stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:15px;margin:20px 0;}
.stat-card{padding:15px;background:#f8f9fa;border:1px solid #dee2e6;border-radius:5px;text-align:center;}
.stat-value{font-size:24px;font-weight:bold;color:#007bff;}
.chart{margin:20px 0;padding:15px;background:white;border:1px solid #ddd;border-radius:5px;}
.bar{height:20px;background:#007bff;margin:2px 0;border-radius:3px;display:flex;align-items:center;padding:0 10px;color:white;font-size:12px;}
</style>
<script>
function showTab(tabName) {
    document.querySelectorAll('.tab').forEach(tab => tab.classList.remove('active'));
    document.querySelectorAll('.tab-content').forEach(content => content.classList.remove('active'));
    document.querySelector('[data-tab="' + tabName + '"]').classList.add('active');
    document.getElementById(tabName).classList.add('active');
}
</script>
</head><body>
<div class="container">
<h1>📦 Inventory Management System</h1>
<div class="tabs">
<a href="#" class="tab {{if ne .ActiveTab "analytics"}}active{{end}}" data-tab="inventory" onclick="showTab('inventory')">Inventory Dashboard</a>
<a href="/?tab=analytics" class="tab {{if eq .ActiveTab "analytics"}}active{{end}}" data-tab="analytics" onclick="showTab('analytics')">📊 Reports & Analytics</a>
</div>
<!-- Inventory Tab -->
<div id="inventory" class="tab-content {{if ne .ActiveTab "analytics"}}active{{end}}">
<div class="form">
<form method="post" action="{{if .EditItem}}/update/{{.EditItem.ID}}{{else}}/add{{end}}">
<h3>{{if .EditItem}}Edit Item{{else}}Add New Item{{end}}</h3>
<div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:10px;">
<div>Name: <input name="name" value="{{if .EditItem}}{{.EditItem.Name}}{{end}}" required></div>
<div>SKU: <input name="sku" value="{{if .EditItem}}{{.EditItem.SKU}}{{end}}" required></div>
<div>Quantity: <input name="quantity" type="number" value="{{if .EditItem}}{{.EditItem.Quantity}}{{end}}" required></div>
<div>Price ($): <input name="price" type="number" step="0.01" value="{{if .EditItem}}{{.EditItem.Price}}{{end}}" required></div>
<div>Supplier: <input name="supplier" value="{{if .EditItem}}{{.EditItem.Supplier}}{{end}}"></div>
<div>Category: <select name="category">
<option value="">Select</option>
<option value="Electronics"{{if .EditItem}}{{if eq .EditItem.Category "Electronics"}} selected{{end}}{{end}}>Electronics</option>
<option value="Office"{{if .EditItem}}{{if eq .EditItem.Category "Office"}} selected{{end}}{{end}}>Office</option>
<option value="Tools"{{if .EditItem}}{{if eq .EditItem.Category "Tools"}} selected{{end}}{{end}}>Tools</option>
<option value="Food"{{if .EditItem}}{{if eq .EditItem.Category "Food"}} selected{{end}}{{end}}>Food</option>
</select></div>
<div>Reorder Point: <input name="reorder" type="number" value="{{if .EditItem}}{{.EditItem.ReorderAt}}{{else}}10{{end}}"></div>
<div>Lead Time (days): <input name="leadtime" type="number" value="{{if .EditItem}}{{.EditItem.LeadTime}}{{else}}7{{end}}"></div>
</div><br>
<button type="submit" class="btn-primary">{{if .EditItem}}Update{{else}}Add{{end}} Item</button>
{{if .EditItem}}<a href="/"><button type="button">Cancel</button></a>{{end}}
</form>
</div>

{{if gt .Stats.LowStock 0}}
<div style="background:#fff3cd;padding:10px;margin:10px 0;border:1px solid #ffeaa7;border-radius:5px;">
<strong>⚠️ Low Stoock Alert!</strong> {{.Stats.LowStock}} item(s) need reordering.
</div>
{{end}}

<form method="get" style="margin:10px 0;">
<input type="text" name="search" placeholder="Search items..." value="{{.SearchTerm}}">
<select name="category" onchange="this.form.submit()">
<option value="">All Categories</option>
<option value="Electronics"{{if eq .Filter "Electronics"}} selected{{end}}>Electronics</option>
<option value="Office"{{if eq .Filter "Office"}} selected{{end}}>Office</option>
<option value="Tools"{{if eq .Filter "Tools"}} selected{{end}}>Tools</option>
<option value="Food"{{if eq .Filter "Food"}} selected{{end}}>Food</option>
</select>
<select name="status" onchange="this.form.submit()">
<option value="">All Status</option>
<option value="low"{{if eq .StatusFilter "low"}} selected{{end}}>Low Stock</option>
<option value="normal"{{if eq .StatusFilter "normal"}} selected{{end}}>Normal</option>
</select>
<button type="submit">Search</button>
</form>

<table>
<tr><th>Name</th><th>SKU</th><th>Qty</th><th>Price</th><th>Supplier</th><th>Category</th><th>Reorder</th><th>Lead Time</th><th>Added</th><th>Status</th><th>Actions</th></tr>
{{range .Items}}
<tr{{if .IsLowStock}} class="low"{{end}}>
<td>{{.Name}}</td><td>{{.SKU}}</td><td>{{.Quantity}}</td><td>${{printf "%.2f" .Price}}</td><td>{{.Supplier}}</td><td>{{.Category}}</td><td>{{.ReorderAt}}</td><td>{{.LeadTime}} days</td><td>{{.DateAdded.Format "01/02/06"}}</td>
<td>{{if .IsLowStock}}<span style="color:red">LOW</span>{{else}}<span style="color:green">OK</span>{{end}}</td>
<td><a href="/edit/{{.ID}}">Edit</a> | <a href="/delete/{{.ID}}" onclick="return confirm('Delete?')">Delete</a></td>
</tr>
{{end}}
</table>
</div>

<!-- Analytics Tab -->
<div id="analytics" class="tab-content {{if eq .ActiveTab "analytics"}}active{{end}}">
<h2>📊 Reports  Analytics</h2>
<div class="stats">
<div class="stat-card"><h4>Total Items</h4><div class="stat-value">{{.Stats.Total}}</div></div>
<div class="stat-card"><h4>Low Stock Items</h4><div class="stat-value" style="color:#dc3545">{{.Stats.LowStock}}</div></div>
<div class="stat-card"><h4>Total Inventory Value</h4><div class="stat-value" style="color:#28a745">${{printf "%.2f" .Stats.TotalValue}}</div></div>
<div class="stat-card"><h4>Categories</h4><div class="stat-value">{{.Stats.Categories}}</div></div>
</div>

<div class="chart">
<h3>📊 Stock Levels by Category</h3>
{{range .CategoryData}}
<div style="margin:10px 0;">
<strong>{{.Name}}</strong>
<div class="bar" style="width:{{.Count}}0%;">{{.Count}} items</div>
</div>
{{end}}
</div>

<div class="chart">
<h3>⏱️ Average Lead Time by Category</h3>
{{range .CategoryData}}
<div style="margin:10px 0;">
<strong>{{.Name}}</strong>
<div class="bar" style="width:{{.AvgLeadTime}}0%;background:#28a745;">{{printf "%.1f" .AvgLeadTime}} days</div>
</div>
{{end}}
</div>
</div>

</div>
</body></html>
`

type PageData struct {
	Items        []*Item
	EditItem     *Item
	Stats        Stats
	CategoryData []CategoryData
	Filter       string
	StatusFilter string
	SearchTerm   string
	ActiveTab    string
}

func (s *Store) add(item *Item) {
	item.ID = s.nextID
	item.DateAdded = time.Now()
	item.IsLowStock = item.Quantity <= item.ReorderAt
	s.items[s.nextID] = item
	s.nextID++
}

func (s *Store) update(id int, item *Item) bool {
	if old, exists := s.items[id]; exists {
		item.ID = id
		item.DateAdded = old.DateAdded
		item.IsLowStock = item.Quantity <= item.ReorderAt
		s.items[id] = item
		return true
	}
	return false
}

func (s *Store) delete(id int) bool {
	if _, exists := s.items[id]; exists {
		delete(s.items, id)
		return true
	}
	return false
}

func (s *Store) getFiltered(categoryFilter, statusFilter, searchTerm string) []*Item {
	var items []*Item
	searchLower := strings.ToLower(searchTerm)
	for _, item := range s.items {
		if categoryFilter != "" && item.Category != categoryFilter {
			continue
		}
		if statusFilter == "low" && !item.IsLowStock {
			continue
		}
		if statusFilter == "normal" && item.IsLowStock {
			continue
		}
		if searchTerm != "" {
			if !strings.Contains(strings.ToLower(item.Name), searchLower) &&
				!strings.Contains(strings.ToLower(item.SKU), searchLower) &&
				!strings.Contains(strings.ToLower(item.Supplier), searchLower) {
				continue
			}
		}
		items = append(items, item)
	}
	return items
}

func (s *Store) getStats() Stats {
	var stats Stats
	categories := make(map[string]bool)
	for _, item := range s.items {
		stats.Total++
		stats.TotalValue += item.Price * float64(item.Quantity)
		if item.IsLowStock {
			stats.LowStock++
		}
		if item.Category != "" {
			categories[item.Category] = true
		}
	}
	stats.Categories = len(categories)
	return stats
}

func (s *Store) getCategoryData() []CategoryData {
	categoryMap := make(map[string][]int)
	for _, item := range s.items {
		category := item.Category
		if category == "" {
			category = "Uncategorized"
		}
		categoryMap[category] = append(categoryMap[category], item.LeadTime)
	}
	var data []CategoryData
	for category, leadTimes := range categoryMap {
		sum := 0
		for _, leadTime := range leadTimes {
			sum += leadTime
		}
		avgLeadTime := float64(sum) / float64(len(leadTimes))
		data = append(data, CategoryData{
			Name:        category,
			Count:       len(leadTimes),
			AvgLeadTime: avgLeadTime,
		})
	}
	return data
}

func parseForm(r *http.Request) (*Item, error) {
	quantity, _ := strconv.Atoi(r.FormValue("quantity"))
	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	reorder, _ := strconv.Atoi(r.FormValue("reorder"))
	leadtime, _ := strconv.Atoi(r.FormValue("leadtime"))

	return &Item{
		Name:      r.FormValue("name"),
		SKU:       r.FormValue("sku"),
		Quantity:  quantity,
		Price:     price,
		Supplier:  r.FormValue("supplier"),
		Category:  r.FormValue("category"),
		ReorderAt: reorder,
		LeadTime:  leadtime,
	}, nil
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("page").Parse(pageTemplate))
	activeTab := r.URL.Query().Get("tab")
	data := PageData{
		Items:        store.getFiltered(r.URL.Query().Get("category"), r.URL.Query().Get("status"), r.URL.Query().Get("search")),
		Stats:        store.getStats(),
		CategoryData: store.getCategoryData(),
		Filter:       r.URL.Query().Get("category"),
		StatusFilter: r.URL.Query().Get("status"),
		SearchTerm:   r.URL.Query().Get("search"),
		ActiveTab:    activeTab,
	}
	tmpl.Execute(w, data)
}

func addHandler(w http.ResponseWriter, r *http.Request) {
	item, _ := parseForm(r)
	store.add(item)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func editHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/edit/"))
	item := store.items[id]
	tmpl := template.Must(template.New("page").Parse(pageTemplate))
	data := PageData{
		Items:        store.getFiltered("", "", ""),
		EditItem:     item,
		Stats:        store.getStats(),
		CategoryData: store.getCategoryData(),
	}
	tmpl.Execute(w, data)
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/update/"))
	item, _ := parseForm(r)
	store.update(id, item)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/delete/"))
	store.delete(id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func main() {
	items := []*Item{
		{Name: "MacBook Pro", SKU: "MBP001", Quantity: 15, Price: 1299.99, Supplier: "Apple Inc", Category: "Electronics", ReorderAt: 5, LeadTime: 7},
		{Name: "Office Chair", SKU: "CHR001", Quantity: 3, Price: 199.99, Supplier: "Furniture Ltd", Category: "Office", ReorderAt: 8, LeadTime: 14},
		{Name: "Wireless Mouse", SKU: "MSE001", Quantity: 25, Price: 29.99, Supplier: "Logitech", Category: "Electronics", ReorderAt: 10, LeadTime: 5},
		{Name: "Safety Helmet", SKU: "HLM001", Quantity: 5, Price: 45.00, Supplier: "Safety Corp", Category: "Tools", ReorderAt: 12, LeadTime: 10},
		{Name: "Coffee Beans", SKU: "COF001", Quantity: 2, Price: 15.99, Supplier: "Coffee Roasters", Category: "Food", ReorderAt: 15, LeadTime: 3},
		{Name: "Laptop Stand", SKU: "LPS001", Quantity: 8, Price: 49.99, Supplier: "Tech Accessories", Category: "Electronics", ReorderAt: 6, LeadTime: 8},
		{Name: "Desk Lamp", SKU: "DLM001", Quantity: 12, Price: 35.99, Supplier: "Office Plus", Category: "Office", ReorderAt: 5, LeadTime: 12},
		{Name: "Drill Set", SKU: "DRL001", Quantity: 4, Price: 89.99, Supplier: "Tool World", Category: "Tools", ReorderAt: 3, LeadTime: 15},
	}
	for _, item := range items {
		store.add(item)
	}

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/add", addHandler)
	http.HandleFunc("/edit/", editHandler)
	http.HandleFunc("/update/", updateHandler)
	http.HandleFunc("/delete/", deleteHandler)

	fmt.Println("Inventory Management System running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
