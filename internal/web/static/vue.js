// Script du tableau de bord : écrit à la main, sans bibliothèque, servi par le
// service lui-même. Il ne fait que de la présentation : la bascule entre la vue
// simple et la vue détaillée, et le tiroir de la vue simple. Toute action sur
// le service reste un formulaire. Sans lui, la page s'affiche dans la vue
// simple et tout reste utilisable.
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
})();
