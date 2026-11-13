// Script du tableau de bord : écrit à la main, sans bibliothèque, servi par le
// service lui-même. Il fait de la présentation : la bascule entre la vue
// simple et la vue détaillée, le tiroir de la vue simple et le repli des
// Groupes. Il fait aussi le glisser-déposer, qui envoie le même formulaire
// que le rangement par ↑/↓ et la liste des Groupes : toute action sur le
// service reste un formulaire. Sans lui, la page s'affiche dans la vue simple
// et tout reste utilisable.
//
// Il ne lit ni n'écrit jamais de HTML : un nom de Store ou de Groupe n'est
// manipulé que comme du texte.
"use strict";

(function () {
  var CLE = "plakar-ui.vue";
  // VUES sont les vues du tableau de bord, la vue par défaut en tête.
  var VUES = ["simple", "detaillee"];
  var racine = document.documentElement;

  // Le navigateur peut refuser le stockage (navigation privée, site bloqué) :
  // la vue n'est alors simplement pas retenue.
  function lire() {
    try {
      return localStorage.getItem(CLE);
    } catch (e) {
      return null;
    }
  }
  function retenir(vue) {
    try {
      localStorage.setItem(CLE, vue);
    } catch (e) {}
  }

  function vueCourante() {
    for (var i = 0; i < VUES.length; i++) {
      if (racine.classList.contains("vue-" + VUES[i])) {
        return VUES[i];
      }
    }
    return VUES[0];
  }

  function appliquer(vue) {
    VUES.forEach(function (v) {
      racine.classList.toggle("vue-" + v, v === vue);
    });
    document.querySelectorAll("[data-vue]").forEach(function (b) {
      b.setAttribute("aria-pressed", String(b.dataset.vue === vue));
    });
  }

  // Le script est chargé dans <head>, avant le corps de la page : la vue
  // retenue s'applique avant le premier affichage, sans clignotement de la
  // vue par défaut, même au rafraîchissement automatique.
  var retenue = lire();
  if (VUES.indexOf(retenue) >= 0) {
    appliquer(retenue);
  }

  function storesOuverts() {
    return document.querySelectorAll("details.store[open]");
  }

  // fermerStoresOuverts ferme les Stores ouverts, sauf garde : les tiroirs de
  // la vue simple, ou les dépliages de la vue détaillée.
  function fermerStoresOuverts(garde) {
    storesOuverts().forEach(function (d) {
      if (d !== garde) {
        d.open = false;
      }
    });
  }

  document.addEventListener("DOMContentLoaded", function () {
    var bascule = document.querySelector("[data-bascule]");
    if (bascule) {
      bascule.hidden = false;
      bascule.addEventListener("click", function (e) {
        var bouton = e.target.closest("[data-vue]");
        if (!bouton) {
          return;
        }
        var vue = bouton.dataset.vue;
        // Les dépliages ouverts de la vue détaillée deviendraient autant de
        // tiroirs empilés.
        if (vue === "simple" && vueCourante() !== "simple") {
          fermerStoresOuverts(null);
        }
        appliquer(vue);
        retenir(vue);
      });
    }
    appliquer(vueCourante());
  });

  // Dans la vue simple, un seul tiroir à la fois : ouvrir un Store ferme les
  // autres. L'événement toggle ne remonte pas, d'où l'écoute en capture.
  document.addEventListener(
    "toggle",
    function (e) {
      var d = e.target;
      if (vueCourante() === "simple" && d.matches && d.matches("details.store") && d.open) {
        fermerStoresOuverts(d);
      }
    },
    true
  );

  // Échap ferme le tiroir, sauf pendant une saisie (le Seuil de retard).
  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape" && vueCourante() === "simple" && !(e.target.closest && e.target.closest("input"))) {
      fermerStoresOuverts(null);
    }
  });

  // ---- Repli des Groupes ----
  // Le navigateur retient les clés des Groupes repliés (data-groupe : le nom
  // du Groupe, ou le mot réservé de « Sans groupe »). Le service n'en sait
  // rien. La clé d'un Groupe renommé ou supprimé reste, sans effet : la page
  // ne montre pas toujours tous les Groupes (focus, « Sans groupe » vide), et
  // ne peut donc pas dire lesquels n'existent plus.
  var CLE_REPLIS = "plakar-ui.replis";
  var SECTION = "section.groupe[data-groupe]";
  var STORE = ".store[data-store]";

  function lireReplis() {
    try {
      var replis = JSON.parse(localStorage.getItem(CLE_REPLIS) || "[]");
      return Array.isArray(replis) ? replis : [];
    } catch (e) {
      return [];
    }
  }
  function retenirReplis(replis) {
    try {
      localStorage.setItem(CLE_REPLIS, JSON.stringify(replis));
    } catch (e) {}
  }

  function replier(section, replie) {
    section.classList.toggle("replie", replie);
    var chevron = section.querySelector("[data-chevron]");
    if (chevron) {
      chevron.textContent = replie ? "▸" : "▾";
      chevron.setAttribute("aria-expanded", String(!replie));
    }
  }

  document.addEventListener("DOMContentLoaded", function () {
    var replis = lireReplis();
    document.querySelectorAll(SECTION).forEach(function (section) {
      section.querySelectorAll("[data-chevron], [data-poignee]").forEach(function (geste) {
        geste.hidden = false;
      });
      // Un Store que le service a ouvert (une action qui vient d'échouer) ne
      // reste pas caché dans un Groupe replié.
      replier(section, replis.indexOf(section.dataset.groupe) >= 0 && !section.querySelector("details.store[open]"));
    });
    // Seule la ligne d'un Store de la config se glisse : un Store disparu ne
    // se range plus, il se déplace par ↑/↓ ou se retire.
    document.querySelectorAll("details" + STORE + " > .ligne").forEach(function (ligne) {
      ligne.draggable = true;
    });
  });

  document.addEventListener("click", function (e) {
    var chevron = e.target.closest && e.target.closest("[data-chevron]");
    if (!chevron) {
      return;
    }
    var section = chevron.closest(SECTION);
    var cle = section.dataset.groupe;
    var replie = !section.classList.contains("replie");
    replier(section, replie);
    var replis = lireReplis().filter(function (k) {
      return k !== cle;
    });
    if (replie) {
      replis.push(cle);
    }
    retenirReplis(replis);
  });

  // ---- Glisser-déposer ----
  // Un Store se glisse par sa ligne (ou sa tuile), un Groupe par sa poignée ⠿.
  // Le dépôt envoie le formulaire de rangement déjà présent dans la page (la
  // liste des Groupes du Store, les ↑/↓ du Groupe), avec en plus le champ
  // « avant » : le Store ou le Groupe avant lequel placer, vide pour la fin.
  // La réponse du service recharge la page. Le sens du dépôt, avant ou après
  // l'élément survolé, suit la mise en page : en grille de plusieurs
  // colonnes, la moitié gauche ou droite ; sinon, la moitié haute ou basse.
  //
  // Les noms ne passent que par dataset et value, jamais par du HTML.
  var glisse = null; // { genre: "store" | "groupe", nom, groupe, el, form }
  var zone = null; // l'élément marqué comme zone de dépôt
  var MARQUES = ["depot-dans", "depot-haut", "depot-bas", "depot-gauche", "depot-droite"];

  function effacerZone() {
    if (zone) {
      MARQUES.forEach(function (classe) {
        zone.classList.remove(classe);
      });
      zone = null;
    }
  }
  function marquerZone(el, classe) {
    if (zone === el && el.classList.contains(classe)) {
      return;
    }
    effacerZone();
    el.classList.add(classe);
    zone = el;
  }

  // enColonnes dit si les frères de el sont posés en grille de plusieurs
  // colonnes (tuiles, colonnes de Groupes) plutôt que les uns sous les autres.
  function enColonnes(el) {
    var colonnes = getComputedStyle(el.parentNode).gridTemplateColumns;
    return !!colonnes && colonnes !== "none" && colonnes.trim().split(/\s+/).length > 1;
  }

  // cote dit où tombe le dépôt sur el : { apres, classe }.
  function cote(el, e) {
    var r = el.getBoundingClientRect();
    if (enColonnes(el)) {
      var droite = e.clientX > r.left + r.width / 2;
      return { apres: droite, classe: droite ? "depot-droite" : "depot-gauche" };
    }
    var bas = e.clientY > r.top + r.height / 2;
    return { apres: bas, classe: bas ? "depot-bas" : "depot-haut" };
  }

  // suivant renvoie le prochain frère de el qui répond au sélecteur, ou null.
  function suivant(el, selecteur) {
    for (var s = el.nextElementSibling; s; s = s.nextElementSibling) {
      if (s.matches(selecteur)) {
        return s;
      }
    }
    return null;
  }

  // avantGroupe donne le champ « avant » qui place un Groupe juste avant
  // cette section : sa clé, ou vide (en dernier) pour « Sans groupe », qui
  // reste dernier, ou pour aucune section.
  function avantGroupe(section) {
    return section && !section.hasAttribute("data-fixe") ? section.dataset.groupe : "";
  }

  // cibleStore dit où tomberait le Store glissé sous ce pointeur : un autre
  // Store (avant ou après lui), ou un Groupe (à la fin). « inutile » signale
  // un dépôt qui le laisserait à sa place.
  function cibleStore(sous, e) {
    var section = sous.closest(SECTION);
    if (!section) {
      return null;
    }
    var store = sous.closest(STORE);
    if (store === glisse.el) {
      return { el: store, inutile: true };
    }
    var groupe = section.dataset.groupe;
    var place, avant;
    if (store) {
      place = cote(store, e);
      var apres = place.apres ? suivant(store, STORE) : store;
      avant = apres ? apres.dataset.store : "";
    } else {
      place = { classe: "depot-dans" };
      avant = "";
      store = section;
    }
    var actuel = suivant(glisse.el, STORE);
    return {
      el: store,
      classe: place.classe,
      inutile: groupe === glisse.groupe && (avant === glisse.nom || avant === (actuel ? actuel.dataset.store : "")),
      champs: { groupe: groupe, avant: avant },
    };
  }

  // cibleGroupe dit où tomberait le Groupe glissé : avant ou après un autre
  // Groupe ; « Sans groupe » reste dernier, un dépôt sur lui place donc en
  // dernier des autres.
  function cibleGroupe(sous, e) {
    var section = sous.closest(SECTION);
    if (!section) {
      return null;
    }
    var place, avant;
    if (section.hasAttribute("data-fixe")) {
      place = { classe: enColonnes(section) ? "depot-gauche" : "depot-haut" };
      avant = "";
    } else {
      place = cote(section, e);
      avant = place.apres ? avantGroupe(suivant(section, SECTION)) : section.dataset.groupe;
    }
    return {
      el: section,
      classe: place.classe,
      inutile: section === glisse.el || avant === glisse.nom || avant === avantGroupe(suivant(glisse.el, SECTION)),
      champs: { avant: avant },
    };
  }

  function cibleDepot(e) {
    if (!glisse) {
      return null;
    }
    var sous = e.target.nodeType === 1 ? e.target : e.target.parentElement;
    if (!sous) {
      return null;
    }
    return glisse.genre === "store" ? cibleStore(sous, e) : cibleGroupe(sous, e);
  }

  // envoyer poste, à l'adresse du formulaire de rangement de l'élément
  // glissé, un formulaire construit avec ces champs : le service répond par
  // une redirection vers la page, ou par la page et son message d'erreur.
  function envoyer(form, champs) {
    var f = document.createElement("form");
    f.method = "post";
    f.action = form.action;
    f.hidden = true;
    Object.keys(champs).forEach(function (nom) {
      var champ = document.createElement("input");
      champ.type = "hidden";
      champ.name = nom;
      champ.value = champs[nom];
      f.appendChild(champ);
    });
    document.body.appendChild(f);
    f.submit();
  }

  document.addEventListener("dragstart", function (e) {
    var source = e.target;
    if (source.nodeType !== 1) {
      return;
    }
    if (source.matches("[data-poignee]")) {
      var section = source.closest(SECTION);
      glisse = { genre: "groupe", nom: section.dataset.groupe, el: section, form: section.querySelector("[data-groupe-entete] form.ordre") };
      // L'image glissée est le Groupe entier, pas la seule poignée.
      e.dataTransfer.setDragImage(section, 16, 16);
    } else if (source.matches("details" + STORE + " > .ligne")) {
      var store = source.parentNode;
      // Le tiroir ouvert de la vue simple ne se glisse pas.
      if (store.open && vueCourante() === "simple") {
        e.preventDefault();
        return;
      }
      glisse = {
        genre: "store",
        nom: store.dataset.store,
        groupe: store.closest(SECTION).dataset.groupe,
        el: store,
        form: store.querySelector("select[name=groupe]").form,
      };
    } else {
      return; // un lien glissé garde son comportement
    }
    e.dataTransfer.effectAllowed = "move";
    e.dataTransfer.setData("text/plain", glisse.nom);
    // Estomper après la capture de l'image glissée, qui sinon serait pâle.
    var el = glisse.el;
    setTimeout(function () {
      el.classList.add("glisse");
    }, 0);
  });

  document.addEventListener("dragover", function (e) {
    var cible = cibleDepot(e);
    if (!cible) {
      effacerZone();
      return;
    }
    e.preventDefault();
    e.dataTransfer.dropEffect = "move";
    if (cible.inutile) {
      effacerZone();
    } else {
      marquerZone(cible.el, cible.classe);
    }
  });

  document.addEventListener("drop", function (e) {
    var cible = cibleDepot(e);
    effacerZone();
    if (!cible) {
      return;
    }
    e.preventDefault();
    if (!cible.inutile) {
      envoyer(glisse.form, cible.champs);
    }
  });

  document.addEventListener("dragend", function () {
    effacerZone();
    if (glisse) {
      glisse.el.classList.remove("glisse");
      glisse = null;
    }
  });
})();
