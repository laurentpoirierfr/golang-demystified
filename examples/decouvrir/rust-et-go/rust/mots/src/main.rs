//! Affiche les dix mots les plus fréquents d'un fichier. Le même programme
//! est écrit en Go dans ../../go/mots, pour comparer les deux langages.

use std::collections::HashMap;
use std::{env, fs, process};

/// Renvoie les n mots les plus fréquents du texte.
fn compte(texte: &str, n: usize) -> Vec<(String, usize)> {
    let mut compte: HashMap<String, usize> = HashMap::new();
    for mot in texte.split(|c: char| !c.is_alphabetic()).filter(|m| !m.is_empty()) {
        *compte.entry(mot.to_lowercase()).or_insert(0) += 1;
    }
    let mut frequences: Vec<(String, usize)> = compte.into_iter().collect();
    frequences.sort_by(|a, b| b.1.cmp(&a.1).then_with(|| a.0.cmp(&b.0)));
    frequences.truncate(n);
    frequences
}

fn main() {
    let args: Vec<String> = env::args().collect();
    if args.len() != 2 {
        eprintln!("usage : mots FICHIER");
        process::exit(2);
    }
    let texte = match fs::read_to_string(&args[1]) {
        Ok(t) => t,
        Err(e) => {
            eprintln!("{e}");
            process::exit(1);
        }
    };
    for (mot, nombre) in compte(&texte, 10) {
        println!("{nombre:8} {mot}");
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn trois_mots() {
        let f = compte("Le gopher creuse. Le crabe pince. Le gopher dort !", 3);
        assert_eq!(
            f,
            vec![("le".to_string(), 3), ("gopher".to_string(), 2), ("crabe".to_string(), 1)]
        );
    }
}
