# VHB_Calendar
###### Outil pour integrer les calendriers des matchs FFHB dans un calendrier partagé du type Google Calendar
Les informations de reference sont recuperées du site:
	http://gesthand-extraction.ff-handball.org/index.php
		=> Menu Competitions
			=> Rencontres à venir
				Bouton "Toutes les semaines"
				=> Export CSV:
	CSV: C'est le fichier qui sera exploité.
 (Ci-dessous, exemple d'enregistrement)

**INSTALLATION:
	Le programme a besoin du fichier credentials.json_secret pour generer le fichier
	token.json ()
   - La 1ere execution va ouvrir le browser en proposant de choisir son compte Google
	- Choix: Mon Club Handball => Authoriser
 	- Copier le code et le copier dans la boite de dialogue => OK
	- le jeton d'autorisation est créé.

**UTILISATION:
	- Lancer l'application
	- Selectionner le fichier CSV
	- La moulinette tourne...et c'est tout.

**BUILD (DEV ONLY):
```
go get github.com/akavel/rsrc
rsrc -manifest vhb_matches_main.manifest -o rsrc.syso
```
Et, pour l'executable:
```
go build
or (if UPX is available)
Go build -ldflags="-s -w" -o VHB_calendar.exe && upx VHB_calendar.exe
```
