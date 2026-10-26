package sim

// Translations holds the simulated agent's texts. Merge them into
// Config.Translations so its commands are described in the right language:
//
//	Translations: i18n.Merge(sim.Translations, myTranslations)
//
// The agent itself answers in the turn's language even without that.
var Translations = map[string]map[string]string{
	"fr": {
		"sim.review.desc":  "Lancer une revue de code (simulée)",
		"sim.review.start": "Revue lancée sur %s.",
		"sim.init.desc":    "Générer le fichier de contexte du projet (simulé)",
		"sim.init.start":   "Analyse du projet pour générer %s.",

		"sim.think1": "L'utilisateur demande : « %s ». Avant de répondre, il me faut du contexte sur le projet %s. " +
			"Je ne connais pas encore son organisation : je commence par lister le dossier, puis je lis un fichier représentatif. " +
			"Je garde une courte liste de tâches pour que l'avancement reste visible. " +
			"Aucun modèle n'est réellement connecté : je dois préciser dans la réponse que ce tour est simulé.",
		"sim.start":      "Je commence par explorer %s pour me faire une idée.",
		"sim.task1":      "Explorer le dossier de travail",
		"sim.task2":      "Lire un fichier représentatif",
		"sim.task3":      "Résumer les constats",
		"sim.sub.title":  "Explorer %s",
		"sim.sub.prompt": "Liste le dossier de travail et lis un fichier représentatif. Rends compte brièvement.",
		"sim.report":     "%d éléments dans %s ; fichier lu : %s.",
		"sim.reportTail": " Je transmets à la session principale.",
		"sim.think2": "Le listing compte %d éléments, dont %d dossiers. J'ai lu %s, ce qui suffit pour un premier résumé. " +
			"Je réponds brièvement et je décris ce que ferait une vraie session ensuite.",
		"sim.done": "Terminé. Ce tour est simulé - aucun modèle n'est connecté, mais les appels d'outils ci-dessus ont réellement été exécutés dans %s.\n\n" +
			"Votre demande : « %s »\n\n" +
			"Le dossier contient %d éléments (%d dossiers, %d fichiers). Dans une vraie session, je planifierais maintenant la modification, éditerais les fichiers et lancerais les tests, en tenant la liste de tâches à jour.",

		"sim.edit.title":  "Ajouter une note en tête de %s",
		"sim.edit.note":   "Note : relu pendant une session simulée.",
		"sim.edit.output": "Simulation : %s n'a pas été modifié sur le disque.",
		"sim.edit.denied": "Modification de %s refusée : je n'y touche pas.",

		"sim.mode.edit":      "Édition",
		"sim.mode.edit.desc": "Demande avant de modifier un fichier, selon les autorisations",
		"sim.mode.plan":      "Plan",
		"sim.mode.plan.desc": "Propose un plan à valider avant d'agir",
		"sim.mode.auto":      "Auto",
		"sim.mode.auto.desc": "Modifie les fichiers sans demander",
		"sim.plan.title":     "Exécuter ce plan ?",
		"sim.plan":           "1. Relire **%s** et le reste du dossier.\n2. Ajouter une note en tête du fichier.\n3. Lancer les tests et résumer les constats.",
		"sim.plan.run":       "Exécuter le plan",
		"sim.plan.keep":      "Continuer à planifier",
		"sim.plan.kept":      "D'accord, je reste en mode plan : rien n'est modifié.",
		"sim.queued":         "J'ai bien reçu votre message pendant le travail : « %s ». Je le prends en compte.",

		"sim.att.think": "L'utilisateur a joint %d fichier(s) : %s. Je lis chacun avant de répondre, " +
			"puis je résume leur contenu. C'est une simulation : le résumé reste générique.",
		"sim.att.read":  "J'ai lu les documents joints - simulation, aucun modèle n'est connecté :",
		"sim.att.ask":   "\n\nVotre demande : « %s »",
		"sim.att.tail":  "\n\nDans une vraie session, je répondrais maintenant à partir de leur contenu. Cliquez sur un nom de fichier pour l'ouvrir dans l'aperçu.",
		"sim.att.pdf":   "Document PDF · %s",
		"sim.att.image": "Image %d×%d · %s",

		"sim.review.think": "L'utilisateur veut une revue de code du dossier de travail. Il n'y a pas de différentiel à examiner : " +
			"je lis donc un fichier source représentatif et je relève quelques améliorations concrètes et sans risque. " +
			"Les constats ci-dessous sont écrits d'avance : c'est une simulation.",
		"sim.review.none": "Rien à relire : le dossier ne contient aucun fichier lisible.",
		"sim.review.out": "Revue de %s (simulée) :\n\n" +
			"- Les fonctions longues qui mêlent mise en page et gestion d'événements gagneraient à être découpées : d'abord les événements, ensuite la mise en page.\n" +
			"- Certains textes sont dupliqués entre fichiers ; un petit bloc de constantes les garderait cohérents.\n\n" +
			"Aucun point bloquant.",

		"sim.init.think": "Pour écrire %s, il me faut une vue d'ensemble du projet : ses fichiers, comment le compiler et ses points d'entrée. " +
			"Je parcours l'arborescence, puis je rédige le fichier. Dans ce simulateur, rien n'est écrit sur le disque.",
		"sim.init.build":  "Compilation",
		"sim.init.layout": "Organisation",
		"sim.init.done":   "%s rédigé - simulation, rien n'a été écrit sur le disque. Cliquez sur le nom du fichier ci-dessus pour l'afficher.",
	},
	"en": {
		"sim.review.desc":  "Run a code review (simulated)",
		"sim.review.start": "Review started on %s.",
		"sim.init.desc":    "Generate the project context file (simulated)",
		"sim.init.start":   "Analyzing the project to generate %s.",

		"sim.think1": "The user is asking: \"%s\". Before answering I need some context about the project in %s. " +
			"I don't know its layout yet, so the first step is to list the directory, then read one representative file. " +
			"I'll keep a short task list so progress stays visible. " +
			"No model is actually connected here, so I should say clearly in the answer that this run is simulated.",
		"sim.start":      "I'll start by looking around %s to get some context.",
		"sim.task1":      "Explore the working directory",
		"sim.task2":      "Read a representative file",
		"sim.task3":      "Summarize findings",
		"sim.sub.title":  "Explore %s",
		"sim.sub.prompt": "List the working directory and read one representative file. Report back briefly.",
		"sim.report":     "%d entries in %s; read %s.",
		"sim.reportTail": " Reporting back to the main session.",
		"sim.think2": "The listing shows %d entries, %d of them folders. I read %s, which is enough for a first summary. " +
			"I'll answer briefly and describe what a real session would do next.",
		"sim.done": "Done. This is a simulated run - no model is connected, but the tool calls above really ran against %s.\n\n" +
			"You asked: \"%s\"\n\n" +
			"The directory has %d entries (%d folders, %d files). In a real session I would now plan the change, edit the files and run the tests, keeping the task list up to date as I go.",

		"sim.edit.title":  "Add a note at the top of %s",
		"sim.edit.note":   "Note: reviewed during a simulated session.",
		"sim.edit.output": "Simulated: %s was not changed on disk.",
		"sim.edit.denied": "Change to %s denied: I'll leave it alone.",

		"sim.mode.edit":      "Edit",
		"sim.mode.edit.desc": "Asks before changing a file, as the permissions say",
		"sim.mode.plan":      "Plan",
		"sim.mode.plan.desc": "Proposes a plan to approve before acting",
		"sim.mode.auto":      "Auto",
		"sim.mode.auto.desc": "Changes files without asking",
		"sim.plan.title":     "Run this plan?",
		"sim.plan":           "1. Read **%s** and the rest of the folder.\n2. Add a note at the top of the file.\n3. Run the tests and summarize the findings.",
		"sim.plan.run":       "Run the plan",
		"sim.plan.keep":      "Keep planning",
		"sim.plan.kept":      "All right, I'll stay in plan mode: nothing is changed.",
		"sim.queued":         "I got your message during the work: \"%s\". I'm taking it into account.",

		"sim.att.think": "The user attached %d file(s): %s. I'll read each one before answering, " +
			"then summarize what they contain. This is a simulation, so the summary stays generic.",
		"sim.att.read":  "I've read the attached document(s) - simulated, no model is connected:",
		"sim.att.ask":   "\n\nYou asked: \"%s\"",
		"sim.att.tail":  "\n\nIn a real session I would now answer from their content. Click a file name to open it in the preview pane.",
		"sim.att.pdf":   "PDF document · %s",
		"sim.att.image": "Image %d×%d · %s",

		"sim.review.think": "The user wants a code review of the working directory. There is no diff to review against, " +
			"so I'll read a representative source file and point out a couple of concrete, low-risk improvements. " +
			"The findings below are canned: this is a simulation.",
		"sim.review.none": "Nothing to review: the directory has no readable file.",
		"sim.review.out": "Review of %s (simulated):\n\n" +
			"- Long functions mixing layout and event handling could be split: handle events first, then lay out.\n" +
			"- Some strings are duplicated across files; a small constants block would keep them consistent.\n\n" +
			"No blocking issue found.",

		"sim.init.think": "To write %s I need an overview of the project: its files, how to build it and the main entry points. " +
			"I'll glob the tree, then draft the file. In this simulator nothing is written to disk.",
		"sim.init.build":  "Build",
		"sim.init.layout": "Layout",
		"sim.init.done":   "%s drafted - simulated, nothing was written to disk. Click the file name above to preview it.",
	},
}
