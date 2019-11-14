/*====================================================================================
* Outil pour integrer les calendriers des matchs FFHB dans un calendrier partagé du
* type Google Calendar
* Les informations de reference sont recuperées du site:
* 	http://gesthand-extraction.ff-handball.org/index.php
*		=> Menu Competitions
*			=> Rencontres à venir
*				Bouton "Toutes les semaines"
*				=> Export CSV:
*	CSV: C'est le fichier qui sera exploité.
* (Ci-dessous, exemple d'enregistrement)
*
*	INSTALLATION:
*	Le programme a besoin du fichier credentials.json_secret pour generer le fichier
*	token.json ()
*   - La 1ere execution va ouvrir le browser en proposant de choisir son compte Google
*	- Choix: Villeneuve Handball => Authoriser
* 	- Copier le code et le copier dans la boite de dialogue => OK
*	- le jeton d'autorisation est créé.
*
*	UTILISATION:
*	- Lancer l'application
*	- Selectionner le fichier CSV
*	- La moulinette tourne...et c'est tout.
* ------------------------------------------------------------------------------------
* TODO: recuperation automatique aprés le login/password de la page:
*	http://gesthand-extraction.ff-handball.org/index.php
======================================================================================
*/

// semaine;"num poule";competition;poule;J;le;horaire;"club rec";"club vis";"club hote";"arb1 designe";"arb2 designe";observateur;delegue;
// "code renc";"nom salle";"adresse salle";CP;Ville;colle;"Coul. Rec";"Coul. Gard. Rec";"Coul. Vis";"Coul. Gard. Vis";
//"Ent. Rec";"Tel Ent. Rec";"Corresp. Rec";"Tel Corresp. Rec";"Ent. Vis";"Tel Ent. Vis";"Corresp. Vis";"Tel Corresp. Vis";"Num rec";"Num vis"

// 2018-39;M610035151;"TEST 2 IGNORE";"Poule 14";2;29/09/2018;15:00:00;"VILLENEUVE HB";"FRONTIGNAN THB";"VILLENEUVE  HANDBALL";;;;;NACCQVW;"COLLEGE LES SALINS";"71 , chemin carrière poissonniere";34750;"VILLENEUVE LES MAGUELONE";"Colle lavable à l'eau uniquement";Bleu;Noir;;;"BOUSIGE THIBAUT";0688310781;"BOUSIGE THIBAUT";0688310781;;;;;6134078;6134029
// ---
// 0: (semaine) 2018-39					// 1: (num poule)M610035151			// 2: (Competition) coupe de france departementale masculine 2018/2019
// 3: (poule) Poule 14					// 4: (Journee) 2					// 5: (date) 29/09/2018
// 6: (horaire) 15:00:00				// 7: (recevante) VILLENEUVE HB		// 8: (visiteur) FRONTIGNAN THB
// 9: (hote) VILLENEUVE HANDBALL		// 10: (arb1)						// 11: (arb2)
// 12: (obs)							// 13: (delegue)					// 14: (code renc) 		NACCQVW
// 15: (nom salle) C OLLEGE LES SALINS	// 16: (adresse salle) 	71 , chemin carrière poissonniere
// 17: (code postal) 	34750			// 18: (ville) VILLENEUVE LES MAGUELONE
// 19: (colle) 	Colle lavable à l'eau uniquement 	// 20: (coul recv) Bleu
// 21: (coul vis) Noir					// 22: (coul gb rec)				// 23: (coul gb vis)
// 24: (ent rec) THE_COACH Bob   		// 25: (tel ent rec) 0688310781		// 26: (corresp rec)  THE_COACH Bob
// 27: (tel corres rec) 0688310781		// 28:	// 29:	// 30:	// 31:
// 32: (num rec)  6134078				// 33: (num visi) 6134029

package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"

	"time"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"

	"strings"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var myDBGTraces = 0 // 0, 1, 2 ...
var myDBGReadOnlyMode = 0

// DEFINES
var Main_Club_ID = "6134078" // Code club
var Entente_MUC_VHB = "VILLENEUVE"

var defVHBCalName = "primary" // TODO
var mColorID = make(map[string]int)

// -----------------------------------
// Usage: ggWeekStart
// mw.te.AppendText(gWeekStart(2018, 1))
//		==> 2018-01-01 00:00:00 +0000 UTC
func gWeekStart(year, week int, loc *time.Location) time.Time {
	// Start from the middle of the year:
	t := time.Date(year, 7, 1, 0, 0, 0, 0, loc)

	// Roll back to Monday:
	if wd := t.Weekday(); wd == time.Sunday {
		t = t.AddDate(0, 0, -6)
	} else {
		t = t.AddDate(0, 0, -int(wd)+1)
	}

	// Difference in weeks:
	_, w := t.ISOWeek()
	t = t.AddDate(0, 0, (week-w)*7)

	return t
}

// //--------------------------------------------------------------------
//--------------------------------------------------------------------
//		2018-01-01 00:00:00 +0000 UTC 2018-01-07 00:00:00 +0000 UTC
func gWeekRange(year, week int) (start, end time.Time) {
	start = gWeekStart(year, week, time.UTC)
	end = start.AddDate(0, 0, 6)
	return
}

// --------------------------------------------------------------------
//--------------------------------------------------------------------
func gPrepareEvent(srv *calendar.Service, strData []string, uCalID string) {

	if myDBGTraces > 2 {
		i := 0
		for v := range strData {
			_ = v
			mw.te.AppendText(fmt.Sprintf("%d: %s\r\n", i, strData[v]))
			i++
		}
		mw.te.AppendText("---------------------------------------\r\n")
	}

	// Verification:
	// 1 - Si pas de club visiteur ou recevant, on ignore
	if (len(strData[32]) == 0) || (len(strData[33]) == 0) {
		// Pas de nom de clubNo data => skip
		return
	}

	// 2 - Le code du club doit être présent dans le club recevant ou visiteur (strData[32] & strData[33] ==> string num Rec et Num Vis)
	if (strData[32] != Main_Club_ID) && (strData[33] != Main_Club_ID) {
		// Si l'id du club n'est ni recev ni visiteur, il s'agit peut etre d'une entente
		// On va donc voir si le nom du club apparait dans un des deux clubs (strData[7] & strData[8])

		// fmt.Printf("\t=>[%s] [%s]\n", strData[7], strData[8])
		//fmt.Printf("\t=>[%v] [%s]\n\n", strings.Contains(Entente_MUC_VHB, strData[7]), strings.Contains(Entente_MUC_VHB, strData[8]))
		// TODO: faire une liste d'identifiant (e.g.: "VILLENEUVE", "VHB", ...)
		if strings.Contains(strData[7], Entente_MUC_VHB) == false && strings.Contains(strData[8], Entente_MUC_VHB) == false {
			// Skip to the next
			return
		}
	}

	mw.te.AppendText(fmt.Sprintf("\r\n==> Prepare Match [%s]\r\n", strData[2]))

	//== Normalization du titre
	compet := strings.ToLower(strData[2])
	compet = strings.Replace(compet, "test", "T", -1)

	compet = strings.Replace(compet, "masculine", "M", -1)
	compet = strings.Replace(compet, "masculin", "M", -1)
	compet = strings.Replace(compet, "feminine", "F", -1)
	compet = strings.Replace(compet, "feminin", "F", -1)
	compet = strings.Replace(compet, "championnat", "", -1)
	compet = strings.Replace(compet, "regional", "Reg.", -1)
	compet = strings.Replace(compet, "honneur", "Hon.", -1)
	compet = strings.Replace(compet, "territorial", "Ter.", -1)
	compet = strings.Replace(compet, "competition", "Comp.", -1)

	summ := fmt.Sprintf("[%s] %s/%s", compet, strData[7], strData[8])
	loc := fmt.Sprintf("%s, %s, %s %s", strData[15], strData[16], strData[17], strData[18])
	desc := fmt.Sprintf("J%s %s", strData[4], strData[2])

	//== Calcule des Dates de debut et fin
	// Si il n'y  pas de date indiquée , on va prendre le samedi de la semaine concernée
	// et on prévoit un horaire de 8:00 du matin :)
	// Durée du créneau: 1h30
	debDate := ""
	endDate := ""
	deb := time.Now()
	hTimeZone := "Europe/Paris"
	locTZ, _ := time.LoadLocation(hTimeZone)
	remind := 1 // Reminder

	if len(strData[5]) == 0 {
		// pas de date, juste une semaine... on va essayer de deviner :)
		// eg: week: "2018-39"
		wYear := 0
		wDay := 0
		fmt.Sscanf(strData[0], "%d-%d", &wYear, &wDay)

		Monday := gWeekStart(wYear, wDay, locTZ)
		Saturday := Monday.AddDate(0, 0, 5)
		deb = Saturday.Add(time.Hour * 8)

		// change few items
		summ = fmt.Sprintf("%s: %s/%s  !!! HORAIRE PAS ENCORE VALIDE !!!", compet, strData[7], strData[8])
		// Par defaut en Go, valeurs à false ...donc les Reminders ne sont pas actifs
		remind = 0
		//event['reminders']['useDefault'] = False;
	} else {
		//There's a date
		when := fmt.Sprintf("%s %s", strData[5], strData[6])
		layout := "02/01/2006 15:04:05" // Expected format
		debx, err := time.Parse(layout, when)
		_ = err
		deb = debx
	}

	// Construction des dates de debut et fin:
	words := strings.Fields(fmt.Sprintf("%s", deb))
	debDate = fmt.Sprintf("%sT%s", words[0], words[1])

	end := deb.Add(time.Minute * 90) //1h30
	words = strings.Fields(fmt.Sprintf("%s", end))
	endDate = fmt.Sprintf("%sT%s", words[0], words[1])

	mw.te.AppendText(fmt.Sprintf("\t==> Horaire: %s / %s\r\n", debDate, endDate))

	//== Color
	//## TODO To fix : modulo  12 for colorid !!!!

	// Si la poule n'existe pas, on ajout une entree dans la map
	if mColorID[strData[3]] == 0 {
		mColorID[strData[3]] = len(mColorID)
	}

	//try:
	//    event['colorId'] = team_list.index(lines['poule'])
	//
	//except ValueError:
	//    team_list.append(lines['poule'])

	//#start my colors from 4
	//colorId := (2 + team_list.index(lines['poule'])) % 12
	//colorId := 4

	strColor := fmt.Sprintf("%d", (2+mColorID[strData[3]])%12)
	//mw.te.AppendText("ColoID: ", strColor)

	//== Tag
	// C'est le marqueur UNIQUE -
	// ATT: si l'entrée est supprimée, on peut la restaurer... par contre si elle est
	// enlevée de la corbeille, le tag continue à exister ==> 403 forbidden
	tag := fmt.Sprintf("a%s%s", strData[1], strData[4]) //  lines['num poule']+lines['J']
	id := strings.ToLower(tag)

	// TODO
	//fix 'id' to be RFC base32 compilant. Should avoid err 400
	// for ch in ['v','w', 'x','y','z']:
	//   if ch in event['id']:
	//     event['id']=event['id'].replace(ch,"p")
	id = strings.Replace(id, "v", "p", -1)
	id = strings.Replace(id, "w", "p", -1)
	id = strings.Replace(id, "x", "p", -1)
	id = strings.Replace(id, "y", "p", -1)
	id = strings.Replace(id, "z", "p", -1)

	// Event construction
	eventX := &calendar.Event{
		Id:          id, // Is UNIQUE !!!
		Summary:     summ,
		Location:    loc,
		Description: desc,
		ColorId:     strColor,

		Start: &calendar.EventDateTime{DateTime: debDate, TimeZone: hTimeZone},
		End:   &calendar.EventDateTime{DateTime: endDate, TimeZone: hTimeZone},
	}

	// Utilisation du reminder par defaut...
	/* TODO attendion sur le creneau de 8h00 du matin... */
	if remind == 1 {
		eventX.Reminders = &calendar.EventReminders{
			Overrides: []*calendar.EventReminder{
				{Method: "popup", Minutes: 60},
			},
			UseDefault:      false,
			ForceSendFields: []string{"UseDefault"},
		}
	}

	//if myDBGTraces > 1 {
	if true {
		log.Println("+++++++++++ DBG Event +++++++++++")
		//TODO Vrai DUMP
		log.Println(fmt.Sprintf("\tID: %s\r\n", id))
		log.Println(fmt.Sprintf("\n\tSummary: %s\n\tloc: %s\n\tdesc: %s \n\ttag: %s", summ, loc, desc, tag))
		log.Println(fmt.Sprintf("\tTZ: %s\r\n", hTimeZone))

		log.Println(fmt.Sprintf("\tHORAIRES  :! deb [%s] / [%s]", debDate, endDate))
		log.Println(fmt.Sprintf("\n+++++++++++++++++++++++++++++"))
	}

	if myDBGReadOnlyMode == 0 {
		time.Sleep(500 * time.Millisecond)
		// Ecriture
		event, err := srv.Events.Insert(uCalID, eventX).Do()

		if err != nil {
			// An error occured:
			log.Println(fmt.Sprintf("==> Error on Insert:\n %v", err))
			switch err.(*googleapi.Error).Code {
			case 400: // TODO
				mw.te.AppendText(fmt.Sprintf("%v\r\n", err))
				log.Println(fmt.Sprintf("%v\r\n", err))
				mw.te.AppendText(fmt.Sprintf("WARNING TODO  ========= (400) IGNORE / PLATEAU TO FIX !!!\r\n"))

			case 403: //TODO
				log.Println(fmt.Sprintf("403: %v\r\n", err))
				log.Println(fmt.Sprintf(" WARNING TODO ======= Time out!!!\r\n"))

			case 409: //  #already exist
				//mw.te.AppendText(fmt.Sprintf("Warning 409: Try to update event %s...", eventX.Id))
				mw.te.AppendText(fmt.Sprintf("Mise à jour event %s...", eventX.Id))
				event, err = srv.Events.Update(uCalID, id, eventX).Do()
				_ = event
				if err == nil {
					mw.te.AppendText(fmt.Sprintf("(%s): OK\r\n", event.Id))
				} else {
					// mw.te.AppendText(fmt.Sprintf("FAIL Result  [%v]\r\n", err))
					mw.te.AppendText(fmt.Sprintf("FAIL Result [%d] \r\n", err.(*googleapi.Error).Code))
				}
			default:
				log.Fatalf("Unable to create event. %v\r\n", err)
				mw.te.AppendText(fmt.Sprintf("Unable to create event. %v\r\n", err))
			}
		} else {
			mw.te.AppendText(fmt.Sprintf("Event created: OK %s\r\n", eventX.HtmlLink))
		}
	} else {
		mw.te.AppendText(fmt.Sprintf("WARNING / Readonly mode:%s\r\n", eventX.Description))
	}
}

//--------------------------------------------------------------------
//--------------------------------------------------------------------
func gProcessCSVFile(srv *calendar.Service, InputCSVfile string, uCalID string) {

	fileIn, err := os.Open(InputCSVfile)
	if err != nil {
		log.Fatal(err)
	}
	defer fileIn.Close()
	skipHeaderDone := 0
	scanner := bufio.NewScanner(fileIn)
	for scanner.Scan() {
		testString := scanner.Text()
		if myDBGTraces > 1 {
			mw.te.AppendText(testString)
			mw.te.AppendText("---")
		}
		testArray := strings.Split(testString, ";")
		i := 0
		for v := range testArray {
			//Clean the datas/remove ""
			if len(testArray[v]) > 0 && testArray[v][0] == '"' {
				testArray[v] = testArray[v][1:]
			}
			if len(testArray[v]) > 0 && testArray[v][len(testArray[v])-1] == '"' {
				testArray[v] = testArray[v][:len(testArray[v])-1]
			}
			i++
		}

		//process to create the event but skip the header
		if skipHeaderDone > 0 {
			gPrepareEvent(srv, testArray, uCalID)
		}
		skipHeaderDone++
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

// ----------------------------------------------------------------------
// gListUpcomingEvents lists all event, starting at a specific date/time
// ----------------------------------------------------------------------
func gListUpcomingEvents(srv *calendar.Service, uCalID string, showdeleted bool) {

	//t := time.Now().Format(time.RFC3339)
	t := "2019-09-01T11:59:50+02:00"

	if myDBGTraces > 1 {
		mw.te.AppendText(fmt.Sprintf("== DBG == gListUpcomingEvents for Cal ID: %s starting @ %s\r\n", uCalID, t))
	}

	events, err := srv.Events.List(uCalID).ShowDeleted(showdeleted).
		SingleEvents(true).TimeMin(t).MaxResults(200).OrderBy("startTime").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
	}

	//Affiche les evenements à venir
	mw.te.AppendText(fmt.Sprintf("\r\nEvenements enregistrés dans le calendrier depuis le %s:\r\n", t))
	if len(events.Items) == 0 {
		mw.te.AppendText("No upcoming events found.\r\n")
	} else {
		for _, item := range events.Items {
			date := item.Start.DateTime
			if date == "" {
				date = item.Start.Date
			}
			mw.te.AppendText(fmt.Sprintf("ID: [%s] - (%s) - ", item.Id, item.Status))
			mw.te.AppendText(fmt.Sprintf("%v (%v)\r\n", item.Summary, date))

		}
	}
}

// ----------------------------------------------------------------------
// gGetCalendarID returns the ID of a wanted calendar
// ----------------------------------------------------------------------
func gGetCalendarID(srv *calendar.Service, calendarName string) string {

	calList, err := srv.CalendarList.List().Do()
	if err != nil {
		log.Fatalf("Unable to retrieve the Calendar list: %v", err)
	}

	if len(calList.Items) == 0 {
		mw.te.AppendText("No calendar found. SHOULD NOT HAPPEND")
	} else {
		for _, iCal := range calList.Items {
			if myDBGTraces > 1 {
				mw.te.AppendText(fmt.Sprintf("ID: [%s] - (%s) \r\n", iCal.Id, iCal.Summary))
			}
			if strings.Compare(calendarName, iCal.Summary) == 0 {
				return iCal.Id
			}
		}
	}
	return ""
}

// ----------------------------------------------------------------------
// Veritable Entry point  :)
// ----------------------------------------------------------------------
func sub_main2(CSVInputFile string, listEventOnly bool, bVerbose bool) {
	myDBGTraces = 0

	if bVerbose == true {
		myDBGTraces = 2
		mw.te.AppendText("(Verbose mode)\r\n")
		return
	}

	mw.te.AppendText(fmt.Sprintf("Fichier à traiter: [%s]\r\n", CSVInputFile))

	// Now, create the serv ref
	srv, err := calendar.New(mw.client)
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v", err)
	}

	// List and Find calendar
	//NOT USED gGetCalendarsList(srv)
	mCalendarID := "primary"
	if strings.Compare(defVHBCalName, "primary") != 0 {
		mCalendarID = gGetCalendarID(srv, defVHBCalName)
	}

	if len(mCalendarID) == 0 {
		log.Fatalf("Unable to retrieve Calendar ID for calname:%s \n%v", defVHBCalName, err)
	}
	if myDBGTraces > 1 {
		mw.te.AppendText(fmt.Sprintf("CalID: %s\r\n", mCalendarID))
	}

	if listEventOnly == true {
		// List all upcoming events for a given agenda
		mw.te.AppendText("Upcoming event deleted")
		gListUpcomingEvents(srv, mCalendarID, true)
	} else {
		gProcessCSVFile(srv, CSVInputFile, mCalendarID)
		mw.te.AppendText("\r\n=== END OF PROCESS ====")

	}
}

//------------------------------------------------------
// MainWindows struct
type MyMainWindow struct {
	*walk.MainWindow
	tabWidget    *walk.TabWidget
	te           *walk.TextEdit
	prevFilePath string
	client       *http.Client
}

var mw = new(MyMainWindow)

//--------------------------------------------------------
func RunMainWindow() error {
	if err := (MainWindow{
		AssignTo: &mw.MainWindow,
		Title:    "VHB Calendar GUI V20190920",
		Size:     Size{640, 480},
		MinSize:  Size{150, 100},
		Layout:   VBox{},
		Children: []Widget{
			PushButton{
				Text: "Ouverture fichier Gesthand Extract",
				OnClicked: func() {
					if err := mw.openMyFile(); err != nil {
						log.Print("OpenDialog:", err)
					} else {
						//mw.te.AppendText(fmt.Sprintf("Fichier à traiter: [%s]\r\n", mw.prevFilePath))
						//===>
						sub_main2(mw.prevFilePath, false /* listEventOnly */, false /* verbose */)

					}
				},
			},

			TextEdit{
				AssignTo: &mw.te,
				ReadOnly: true,
				VScroll:  true,
			},
		},
	}).Create(); err != nil {
		log.Fatal(err)
		return err
	}

	mw.Run()

	return nil
}

// --------------------------------------------------------------------
func (mw *MyMainWindow) openMyFile() error {
	dlg := new(walk.FileDialog)

	dlg.FilePath = mw.prevFilePath
	dlg.Filter = "Fichier Gesthand Extract (*.csv)|*.csv"
	dlg.Title = "Selection du fichier Gesthand"

	if ok, err := dlg.ShowOpen(mw); err != nil {
		return err
	} else if !ok {
		return nil
	}

	// Store file name
	mw.prevFilePath = dlg.FilePath

	return nil
}

// *****************************************************************************
// ENTRY POINT
// *****************************************************************************
func main() {
	mw.client = VHB_GetCredentials() //Extern code

	// Call mainw()
	if err := RunMainWindow(); err != nil {
		log.Fatal(err)
	}
}
