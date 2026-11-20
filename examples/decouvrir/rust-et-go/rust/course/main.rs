// La course de données de ../../go/course, écrite en Rust : deux fils
// d'exécution modifient le même compteur sans synchronisation. Ce programme
// NE COMPILE PAS, et c'est voulu : le compilateur refuse la course.
//
//   rustc course/main.rs
use std::thread;

fn main() {
    let mut compteur = 0;
    thread::scope(|s| {
        for _ in 0..2 {
            s.spawn(|| {
                for _ in 0..100_000 {
                    compteur += 1;
                }
            });
        }
    });
    println!("{compteur}");
}
