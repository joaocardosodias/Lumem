package adapter_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/g04/backend/internal/adapter/parser"
	"github.com/xuri/excelize/v2"
)

func TestAlarmParser_CSV_EmptyError(t *testing.T) {
	p := parser.NewDefaultAlarmParser()
	if _, err := p.ParseAlarms(strings.NewReader(""), "csv"); err == nil {
		t.Fatalf("esperava erro para arquivo vazio")
	}
}

func TestAlarmParser_CSV_OnlyHeaderError(t *testing.T) {
	p := parser.NewDefaultAlarmParser()
	if _, err := p.ParseAlarms(strings.NewReader("tag,estacao,porta,alarme,horario_alarme\n"), "csv"); err == nil {
		t.Fatalf("esperava erro para apenas cabecalho")
	}
}

func TestAlarmParser_CSV_MissingRequiredColumns(t *testing.T) {
	p := parser.NewDefaultAlarmParser()
	csvData := "col1,col2\nval1,val2\n"
	if _, err := p.ParseAlarms(strings.NewReader(csvData), "csv"); err == nil {
		t.Fatalf("esperava erro para colunas ausentes")
	}
}

func TestAlarmParser_CSV_Success(t *testing.T) {
	p := parser.NewDefaultAlarmParser()
	csvData := "tag,estacao,porta,alarme,horario_alarme,tempo_normalizacao_seg,horario_normalizacao,plataforma\n" +
		"SP04-6-PSD-PIN-P1-PP01,PIN,1.0,Obstaculo,2026-09-01 10:00:00,10.0,2026-09-01 10:00:10,P1\n" +
		"SP04-6-PSD-PIN-P2-PP02,PIN,2,Falha,2026-09-01 10:05:00,5,2026-09-01 10:05:05,\n" +
		"TAG-SEM-PLAT,HIG,3,Alerta,2026-09-01 10:10:00,,,\n"

	records, err := p.ParseAlarms(strings.NewReader(csvData), "csv")
	if err != nil || len(records) != 3 {
		t.Fatalf("esperava 3 registros, erro: %v", err)
	}

	if records[0].Platform != "P1" || records[0].DoorNumber != 1 || records[0].DurationSeconds != 10 {
		t.Errorf("registro 0 incorreto: %+v", records[0])
	}
	if records[1].Platform != "P2" || records[1].DoorNumber != 2 || records[1].DurationSeconds != 5 {
		t.Errorf("registro 1 incorreto: %+v", records[1])
	}
	if records[2].Platform != "" || records[2].DoorNumber != 3 || records[2].DurationSeconds != 0 {
		t.Errorf("registro 2 incorreto: %+v", records[2])
	}
}

func TestMaintenanceParser_CSV_EmptyError(t *testing.T) {
	p := parser.NewDefaultMaintenanceParser()
	if _, err := p.ParseMaintenance(strings.NewReader(""), "csv", ""); err == nil {
		t.Fatalf("esperava erro para arquivo vazio")
	}
}

func TestMaintenanceParser_CSV_MissingStartDateError(t *testing.T) {
	p := parser.NewDefaultMaintenanceParser()
	csvData := "ordem,tipo_de_ordem,estacao\n123,ZM03,PIN\n"
	if _, err := p.ParseMaintenance(strings.NewReader(csvData), "csv", ""); err == nil {
		t.Fatalf("esperava erro para data de inicio ausente")
	}
}

func TestMaintenanceParser_CSV_Success(t *testing.T) {
	p := parser.NewDefaultMaintenanceParser()
	csvData := "ordem,tipo_de_ordem,local_de_instalacao,estacao,inicio_avaria_datetime,fim_avaria_datetime,duracao_avaria_horas,descricao,sintoma,causa,acao\n" +
		"ORD-1,ZM01,PLATAFORMA 1,PIN,2026-09-01 10:00:00,2026-09-01 11:30:00,\"1,5\",Troca,Defeito,Desgaste,Substituicao\n" +
		"ORD-2,,PLATAFORMA 2,HIG,2026-09-01 12:00:00,,2.0,Ajuste,,,,,\n"

	records, err := p.ParseMaintenance(strings.NewReader(csvData), "csv", "PADRAO")
	if err != nil || len(records) != 2 {
		t.Fatalf("esperava 2 registros, erro: %v", err)
	}

	if records[0].OrderType != "PADRAO" || records[0].DurationHours != 1.5 {
		t.Errorf("registro 0 incorreto: %+v", records[0])
	}

	recordsFallback, err := p.ParseMaintenance(strings.NewReader(csvData), "csv", "")
	if err != nil {
		t.Fatalf("falha ao processar sem default order type: %v", err)
	}
	if recordsFallback[0].OrderType != "ZM01" || recordsFallback[1].OrderType != "MANUTENCAO" {
		t.Errorf("fallback de order type incorreto: rec0=%s, rec1=%s", recordsFallback[0].OrderType, recordsFallback[1].OrderType)
	}
}

func TestParser_Excel(t *testing.T) {
	f := excelize.NewFile()
	sheet := "Sheet1"
	_ = f.SetCellValue(sheet, "A1", "tag")
	_ = f.SetCellValue(sheet, "B1", "estacao")
	_ = f.SetCellValue(sheet, "C1", "porta")
	_ = f.SetCellValue(sheet, "D1", "alarme")
	_ = f.SetCellValue(sheet, "E1", "horario_alarme")

	_ = f.SetCellValue(sheet, "A2", "SP04-6-PSD-PIN-P1-PP01")
	_ = f.SetCellValue(sheet, "B2", "PIN")
	_ = f.SetCellValue(sheet, "C2", "1")
	_ = f.SetCellValue(sheet, "D2", "Obstaculo")
	_ = f.SetCellValue(sheet, "E2", "2026-09-01 10:00:00")

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("falha ao gerar excel em memoria: %v", err)
	}

	p := parser.NewDefaultAlarmParser()
	records, err := p.ParseAlarms(bytes.NewReader(buf.Bytes()), "xlsx")
	if err != nil {
		t.Fatalf("falha ao ler excel de alarmes: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("esperava 1 registro lido do excel, obteve %d", len(records))
	}

	_, errCorrupt := p.ParseAlarms(strings.NewReader("not-a-valid-zip-excel"), ".xlsx")
	if errCorrupt == nil {
		t.Fatalf("esperava erro para excel corrompido")
	}
}

type panicReader struct{}

func (p *panicReader) Read(b []byte) (int, error) {
	panic("panic forced")
}

func TestParser_PanicRecovery(t *testing.T) {
	alarmParser := parser.NewDefaultAlarmParser()
	_, err := alarmParser.ParseAlarms(&panicReader{}, "csv")
	if err == nil || !strings.Contains(err.Error(), "panico recuperado") {
		t.Fatalf("esperava recuperacao de panico em alarm parser, obteve %v", err)
	}

	maintParser := parser.NewDefaultMaintenanceParser()
	_, err = maintParser.ParseMaintenance(&panicReader{}, "csv", "")
	if err == nil || !strings.Contains(err.Error(), "panico recuperado") {
		t.Fatalf("esperava recuperacao de panico em maintenance parser, obteve %v", err)
	}
}

func TestMaintenanceParser_OnlyHeader(t *testing.T) {
	p := parser.NewDefaultMaintenanceParser()
	_, err := p.ParseMaintenance(strings.NewReader("ordem,inicio_avaria_datetime\n"), "csv", "")
	if err == nil {
		t.Fatalf("esperava erro de arquivo vazio ou sem linhas de dados")
	}
}

func TestMaintenanceParser_CorruptedExcel(t *testing.T) {
	p := parser.NewDefaultMaintenanceParser()
	_, err := p.ParseMaintenance(strings.NewReader("not-a-valid-zip-excel"), ".xlsx", "")
	if err == nil {
		t.Fatalf("esperava erro para excel corrompido em maintenance parser")
	}
}

