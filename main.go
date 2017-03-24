package main

import (
	"flag"
	"fmt"
	"github.com/chapzin/parse-efd-fiscal/SpedDB"
	"github.com/chapzin/parse-efd-fiscal/SpedRead"
	"github.com/chapzin/parse-efd-fiscal/config"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"time"
	"github.com/chapzin/GoInventario/Schema"
	"strconv"
	"sync"
	"github.com/chapzin/GoInventario/Tools"
	"github.com/chapzin/parse-efd-fiscal/Controllers"
	"github.com/fatih/color"
)

var schema = flag.Bool("schema", false, "Recria as tabelas")
var importa = flag.Bool("importa",false,"Importa os xmls e speds ")
var inventario = flag.Bool("inventario",false,"Fazer processamento do inventario")

func init() {
	flag.Parse()
	cfg := new(config.Configurador)
	config.InicializaConfiguracoes(cfg)
}

func main() {
	dialect, err := config.Propriedades.ObterTexto("bd.dialect")
	conexao, err := config.Propriedades.ObterTexto("bd.conexao")
	db, err := gorm.Open(dialect, conexao)
	if err != nil {
		fmt.Println("Falha ao abrir conexão. dialect=?, Linha de Conexao=?", dialect, conexao)
		return
	}

	if *schema {
		// Recria o Schema do banco de dados
		SpedDB.Schema(*db)
	}

	if *importa {
		// Lendo todos arquivos da pasta speds
		fmt.Println("Iniciando processamento ", time.Now())
		SpedRead.RecursiveSpeds("./speds", dialect, conexao)
		// Pega cada arquivo e ler linha a linha e envia para o banco de dados
		fmt.Println("Final processamento ", time.Now())
		var msg string
		fmt.Scanln(&msg)
	}

	if *inventario {
		// Processa o inventário
		fmt.Println(time.Now())
		var wg sync.WaitGroup
		db, err := gorm.Open("mysql", "root:123@/auditoria2?charset=utf8&parseTime=true")
		//db.LogMode(true)
		Tools.CheckErr(err)
		schema := flag.Bool("schema", false, "Recria as tabelas")
		ano := flag.Int("ano", 0, "Ano do processamento do inventário")
		defer db.Close()
		flag.Parse()
		if *schema {

			Schema.DropSchema(*db)
			Schema.CreateSchema(*db)
		}
		if *ano == 0 {
			fmt.Println("Favor informar o ano que deseja processar. Exemplo -ano=2017")
			return
		} else if *ano <= 2011 {
			fmt.Println("Favor informar um ano maior que 2011")
			return
		} else if *ano <= 999 {
			fmt.Println("Favor informar o ano com 4 digitos. Exemplo 2017")
			return
		}
		anoString := strconv.Itoa(*ano)

		wg.Add(2)
		go Controllers.ProcessarFatorConversao(*db, &wg)
		go Controllers.DeletarItensNotasCanceladas(*db, "2012-01-01", "2016-12-31", &wg)
		wg.Wait()
		wg.Add(2)
		go Controllers.PopularReg0200(*db, &wg)
		go Controllers.PopularItensXmls(*db, &wg)
		wg.Wait()
		wg.Add(4)
		// Podem rodar em concorrencia
		go Controllers.PopularInventario("inicial", *ano, &wg)
		go Controllers.PopularInventario("final", *ano, &wg)
		go Controllers.PopularEntradas(anoString, &wg)
		go Controllers.PopularSaidas(anoString, &wg)
		wg.Wait()
		// fim de rodar em concorrencia
		// Quando finalizar todas essas deve rodar o processar diferencas
		Controllers.ProcessarDiferencas(*db)
		fmt.Println(time.Now())
		color.Green("TERMINOUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUU")
		time.Sleep(60 * time.Second)
	}


}
