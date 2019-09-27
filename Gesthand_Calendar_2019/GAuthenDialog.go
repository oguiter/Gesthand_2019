//===========================================================
// Code de gestion de l'authentification
//===========================================================

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	//"strings"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

// DEFINES
// Get credential from https://console.developers.google.com/apis/credentials?authuser=1&project=omega-keep-181716
var defSecretCreds_from_file = "credentials.json_secret"
var defUserToken = "token.json"

//----------------------------------------------------------------
// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	tokFile := defUserToken
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb_UI(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

//----------------------------------------------------------------
// Helper to open browser...
func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

//----------------------------------------------------------------
// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb_UI(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Merci d'ouvrir le lien suivant dans votre navigateur puis de saisir "+
		"ci dessous le code d'autorisation indiqué: \n%v\r\n", authURL)

	//Open the associated browser
	openbrowser(authURL)

	// Prepare to display Get Code Dialog
	dlgdata := new(MyDialogData)
	// Call the dialog
	cmd, err := RunAskAuthenticationDialog(nil, dlgdata)
	if err != nil {
		fmt.Println(err)
	} else if cmd == walk.DlgCmdCancel { // Cancel
		log.Fatalf("Unable to run without authentication code")
		return nil
	} else if cmd == walk.DlgCmdNone { // 右上xクリック
		log.Fatalf("Unable to run without authentication code")
		return nil
	}

	// Got something !
	fmt.Printf("Authentication code: [%s]", dlgdata.msg)

	tok, err := config.Exchange(oauth2.NoContext, dlgdata.msg)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

//----------------------------------------------------------------
// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

//----------------------------------------------------------------
// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	log.Printf("Saving credential file to: %s\r\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	defer f.Close()
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	json.NewEncoder(f).Encode(token)
}

//================================dialog
//================================
type MyDialogData struct {
	msg string
}
type MyDialogWindow struct {
	dlg      *walk.Dialog
	edit     *walk.LineEdit
	acceptPB *walk.PushButton
	cancelPB *walk.PushButton
}

//=======================================

func RunAskAuthenticationDialog(owner walk.Form, dlgdata *MyDialogData) (int, error) {

	mydlg := new(MyDialogWindow)

	MYDLG := Dialog{
		AssignTo:      &mydlg.dlg,
		Title:         "Google Agenda Authentication Code",
		DefaultButton: &mydlg.acceptPB,
		CancelButton:  &mydlg.cancelPB,
		MinSize:       Size{300, 100},
		Layout:        VBox{},
		Children: []Widget{
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Label{
						Text: "Code d'authentification ?",
					},
					LineEdit{
						AssignTo: &mydlg.edit,
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{
						AssignTo: &mydlg.acceptPB,
						Text:     "OK",
						OnClicked: func() {
							dlgdata.msg = mydlg.edit.Text()
							mydlg.dlg.Accept()
						},
					},
					PushButton{
						AssignTo:  &mydlg.cancelPB,
						Text:      "Cancel",
						OnClicked: func() { mydlg.dlg.Cancel() },
					},
				},
			},
		},
	}

	return MYDLG.Run(owner)
}

//------------------------------------------------------
func VHB_GetCredentials() *http.Client {

	// Set my secret credentials
	//
	// NOT LOCAL: b := []byte(defSecretCreds)

	b, err := ioutil.ReadFile(defSecretCreds_from_file)
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	// Readonly config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	config, err := google.ConfigFromJSON(b, calendar.CalendarScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	return getClient(config)
}
