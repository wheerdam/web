package main

import (
	"os"
	"log"
	"fmt"
    "crypto/tls"
    "net/http"
	"database/sql"
	"golang.org/x/crypto/acme/autocert"
	"github.com/wheerdam/netutil"
	"github.com/wheerdam/inventory"	
	"github.com/icza/session"
	//"bbi/inventory"
)

var db *sql.DB
var dirTemplates string
var dirStatic string

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}
	switch os.Args[1] {
	case "users":
		if len(os.Args) < 4 {
			usersUsage()
			return
		}
		handleUserOps()
	case "inventory":
		if len(os.Args) < 4 {
			inventoryUsage()
			return
		}
		handleDbOps()
	case "serve":
		if len(os.Args) < 5 {
			serveUsage()
			return
		}
		dirTemplates = os.Args[3]
		dirStatic = os.Args[4]
		indexHttps := 0
		indexHttpsLe := 0
		indexInv := 0
		if len(os.Args) >= 8 {
			if os.Args[5] == "https" {
				indexHttps = 5
			} else if os.Args[5] == "https-le" {
				indexHttpsLe = 5
			} else if os.Args[5] == "inventory" {
				indexInv = 5
			} else {
				serveUsage()
				return
			}
			if len(os.Args) == 13 {
				if indexInv > 0 && os.Args[10] == "https" {
					indexHttps = 10
				} else if indexInv > 0 && os.Args[10] == "https-le" {
					indexHttpsLe = 10
				} else if os.Args[8] == "inventory" {
					indexInv = 8
				} else {
					serveUsage()
					return
				}
			} else if len(os.Args) > 10 {
				fmt.Println("bah")
				serveUsage()
				return
			}
		}
		var err error
		db, err = netutil.OpenPostgresDBFromConfig(os.Args[2])
		if err != nil {
			log.Fatal(err)
		}
		if indexInv > 0 {
			err = inventory.Install(os.Args[indexInv+1],
				os.Args[indexInv+2], os.Args[indexInv+3],
				os.Args[indexInv+4], db)//os.Args[2])
			if err != nil {
				log.Fatal(err)
			}
		}
		http.HandleFunc("/", indexHandler)
		fs := http.FileServer(http.Dir(os.Args[4]))
		http.Handle("/s/", http.StripPrefix("/s/", fs))
		if indexHttpsLe > 0 {
			certManager := autocert.Manager{
				Prompt:     autocert.AcceptTOS,
				HostPolicy: autocert.HostWhitelist(os.Args[indexHttpsLe+1]),
				Cache:      autocert.DirCache(os.Args[indexHttpsLe+2]),
			}		
			server := &http.Server{
				Addr: ":443",
				TLSConfig: &tls.Config{
					GetCertificate: certManager.GetCertificate,
				},
			}
			go http.ListenAndServe(":80", http.HandlerFunc(redirectHTTP))
			err := server.ListenAndServeTLS("", "")
			if err != nil {
				log.Fatal("ListenAndServeTLS: ", err)
			}
		} else if indexHttps > 0 {
			go http.ListenAndServe(":80", http.HandlerFunc(redirectHTTP))
			err := http.ListenAndServeTLS(":443", os.Args[indexHttps+1], 
					os.Args[indexHttps+2], nil)
			if err != nil {
				log.Fatal("ListenAndServeTLS: ", err)
			}
		} else {
			session.Global.Close()
			session.Global = session.NewCookieManagerOptions(session.NewInMemStore(), &session.CookieMngrOptions{AllowHTTP: true})
			err := http.ListenAndServe(":80", nil)
			if err != nil {
				log.Fatal("ListenAndServe: ", err)
			}
		}
	default:
		usage()
	}    
}

func usersUsage() {
	fmt.Println("  users [users-file] list")
	fmt.Println("                     add [username] [password]")
	fmt.Println("                     delete [username]")
	fmt.Println("                     test-login [username] [password]")
}

func serveUsage() {
	fmt.Println("  serve [db-config] [templates-dir] [static-dir]")
	fmt.Println("          (https-le [domain-name] [cert-dir])")
	fmt.Println("          (https [cert] [key])")
	fmt.Println("          (inventory [prefix] [users-file] [templates-dir] [static-dir])")
}

func inventoryUsage() {
	fmt.Println("  inventory [db-config] create-default-config")
	fmt.Println("                        create-tables")
	fmt.Println("                        delete-tables")
	fmt.Println("                        export-items [output-file]")
	fmt.Println("                        export-inventory [output-file]")
	fmt.Println("                        import-items [input-file]")
	fmt.Println("                        import-inventory [input-file]")
	fmt.Println("                        list-items")
	fmt.Println("                        list-inventory")
}

func usage() {
	fmt.Println("usage: web [command]\n")
	fmt.Println("commands:")
	usersUsage()
	fmt.Println()
	inventoryUsage()
	fmt.Println()	
	serveUsage()
	fmt.Println()
}

func handleUserOps() {
	path := os.Args[2]
	command := os.Args[3]
	params := os.Args[4:]
	users := netutil.NewUsers()
	err := users.LoadFromFile(path)
	if err != nil {
		log.Fatal(err)
	}

	switch command {
	case "list":
		userList := users.GetList()
		fmt.Println(len(userList), "users: ")
		for i := range userList {
			fmt.Print(i, " " + userList[i] + "\n")
		}
	case "add":
		if len(params) != 2 {
			fmt.Println("usage: inventory users [users-file] add [name] [password]")
			return
		}
		fmt.Println("Adding user '" + params[0] + "'")
		err := users.Add(params[0], params[1])
		if err != nil {
			log.Fatal(err)
		}
	case "test-login":
		if len(params) != 2 {
			fmt.Println("usage: inventory users [users-file] test-login [name] [password]")
			return
		}
		err := users.Login(params[0], params[1])
		if err != nil {
			log.Fatal(err)
		} else {
			fmt.Println("Login Successful")
		}
	case "delete":
		if len(params) != 1 {
			fmt.Println("usage: inventory users [users-file] delete [name]")
			return
		}
		users.Delete(params[0])
	default:
		fmt.Println(command, "is an invalid subcommand\n")
		usersUsage()
		return
	}
	
	err = users.SaveToFile(path)
	if err != nil {
		log.Fatal(err)
	}
}

func handleDbOps() {
	path := os.Args[2]
	command := os.Args[3]
	params := os.Args[4:]
	
	if command == "create-default-config" {
		if len(params) != 0 {
			fmt.Println("usage: inventory db [db-config] create-default-config")
			return
		}
		defaultStr := []byte("bbiinv bbipassword localhost 5432 bbiinvdb disable\n")
		f, err := os.Create(path)
		if err != nil {
			log.Fatal(err)
		}
		_, err = f.Write(defaultStr)
		if err != nil {
			log.Fatal(err)
		}
		f.Close()
		return;
	}
	var err error
	db, err := netutil.OpenPostgresDBFromConfig(path)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	
	switch command {
	case "create-tables":		
		err := inventory.CreateTables(db)
		if err != nil {
			log.Fatal(err)
		}
	case "delete-tables":
		err := inventory.DeleteTables(db)
		if err != nil {
			log.Fatal(err)
		}
	case "import-inventory":
		if len(params) != 1 {
			fmt.Println("usage: inventory db [db-config] import-inventory [input-file]")
			return
		}
		file, err := os.Open(params[0])
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		err = inventory.ImportInventory(file, db)
		if err != nil {
			log.Fatal(err)
		}
	case "import-items":
		if len(params) != 1 {
			fmt.Println("usage: inventory db [db-config] import-items [input-file]")
			return
		}
		file, err := os.Open(params[0])
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		err = inventory.ImportItems(file, db)
		if err != nil {
			log.Fatal(err)
		}
	case "list-inventory":
		rows, err := db.Query("select * from inventory")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("itemID\t Qty\t Location")
		fmt.Println("------\t ---\t --------")
		for rows.Next() {
			var entry inventory.InventoryEntry
			err := rows.Scan(&entry.Serial,
				&entry.ItemID, &entry.Location, &entry.Quantity)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(entry.ItemID, "\t", entry.Quantity, "\t",
						entry.Location)
		}
	case "list-items":
		rows, err := db.Query("select * from items")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		fmt.Println("itemID\t descriptive_name")
		fmt.Println("------\t ----------------")
		for rows.Next() {
			var item inventory.Item
			err := rows.Scan(
				&item.Serial, &item.ItemID, &item.Descriptive_name,
				&item.Model_number, &item.Manufacturer, 
				&item.Type, &item.Subtype,
				&item.Phys_description,
				&item.DatasheetURL, &item.ProductURL,
				&item.Seller1URL, &item.Seller2URL,
				&item.Seller3URL, &item.UnitPrice, &item.Notes,
				&item.Value,
				)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(item.ItemID, "\t",
						item.Descriptive_name)
		}
	case "export-inventory":
		if len(params) != 1 {
			fmt.Println("usage: inventory db [db-config] export-inventory [output-file]")
			return
		}
		f, err := os.Create(params[0])
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		err = inventory.ExportInventory(f, db)
		if err != nil {
			log.Fatal(err)
		}
	case "export-items":
		if len(params) != 1 {
			fmt.Println("usage: inventory db [db-config] export-items [output-file]")
			return
		}
		f, err := os.Create(params[0])
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		err = inventory.ExportItems(f, db)				
		if err != nil {
			log.Fatal(err)
		}
	default:
		fmt.Println(command, "is an invalid subcommand\n")
		inventoryUsage()
	}
}
