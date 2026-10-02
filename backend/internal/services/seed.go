package services

import (
	"aulaquest/internal/models"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"os"
)

func Seed(db *gorm.DB) error {
	if os.Getenv("APP_ENV") == "production" {
		return fmt.Errorf("synthetic seed is forbidden in production")
	}
	sp, tp := os.Getenv("SEED_STUDENT_PASSWORD"), os.Getenv("SEED_TEACHER_PASSWORD")
	if len(sp) < 12 || len(sp) > 72 || len(tp) < 12 || len(tp) > 72 {
		return fmt.Errorf("configure both SEED passwords with 12..72 bytes")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		ids := map[string]int64{}
		for _, item := range []struct{ name, alias, role, password string }{{"luna", "Luna", "student", sp}, {"sol", "Sol", "student", sp}, {"profe", "Profe Alex", "teacher", tp}} {
			hash, e := bcrypt.GenerateFromPassword([]byte(item.password), 12)
			if e != nil {
				return e
			}
			if e = tx.Exec("INSERT INTO users(username,alias,role,password_hash) VALUES (?,?,?,?) ON CONFLICT(username) DO NOTHING", item.name, item.alias, item.role, string(hash)).Error; e != nil {
				return e
			}
			var u models.User
			if e = tx.Where("username=?", item.name).Take(&u).Error; e != nil {
				return e
			}
			ids[item.name] = u.ID
			if item.role == "student" {
				if e = tx.Exec("INSERT INTO gamification_profiles(user_id) VALUES (?) ON CONFLICT DO NOTHING", u.ID).Error; e != nil {
					return e
				}
			}
		}
		if e := tx.Exec("INSERT INTO teacher_students(teacher_id,student_id) VALUES (?,?) ON CONFLICT DO NOTHING", ids["profe"], ids["luna"]).Error; e != nil {
			return e
		}
		for _, c := range []struct{ slug, title, description, icon string }{{"creative-coding", "Exploradores del código", "Tu primera aventura entre ideas, instrucciones y pequeños descubrimientos.", "🚀"}, {"reading-worlds", "Historias que despiertan", "Lee, imagina y descubre mundos con tus propias palabras.", "🌱"}} {
			if e := tx.Exec("INSERT INTO courses(slug,title,description,icon) VALUES (?,?,?,?) ON CONFLICT(slug) DO NOTHING", c.slug, c.title, c.description, c.icon).Error; e != nil {
				return e
			}
			var course models.Course
			if e := tx.Where("slug=?", c.slug).Take(&course).Error; e != nil {
				return e
			}
			if e := tx.Exec("INSERT INTO course_staff(course_id,user_id) VALUES (?,?) ON CONFLICT DO NOTHING", course.ID, ids["profe"]).Error; e != nil {
				return e
			}
			for _, name := range []string{"luna", "sol"} {
				if e := tx.Exec("INSERT INTO enrollments(user_id,course_id) VALUES (?,?) ON CONFLICT DO NOTHING", ids[name], course.ID).Error; e != nil {
					return e
				}
			}
			if e := tx.Exec("INSERT INTO modules(course_id,title,position) VALUES (?,'Primeros descubrimientos',1) ON CONFLICT(course_id,position) DO NOTHING", course.ID).Error; e != nil {
				return e
			}
			var m models.Module
			if e := tx.Where("course_id=? AND position=1", course.ID).Take(&m).Error; e != nil {
				return e
			}
			lessons := []struct{ title, desc, kind, conf string }{{"Cada aventura empieza con una idea", "Lee a tu ritmo. Aquí puedes volver a intentarlo siempre.", "reading", `{"body":"Luna quería llegar a la biblioteca del bosque. Primero miró el mapa, después eligió un camino y finalmente dio el primer paso. Un plan es una lista de pasos: también lo llamamos algoritmo.\n\nPiensa en algo que haces cada mañana. ¿Qué paso va primero? ¿Qué pasaría si cambias el orden? Aprender empieza con una pregunta."}`}, {"Tu laboratorio de ideas", "Escribe instrucciones y conoce una consola simulada.", "code", `{"language":"javascript","starter":"// Escribe tu primera idea\nconsole.log('¡Hola, aventura!');"}`}, {"Un reto para explorar", "Una actividad interactiva que tu docente puede configurar.", "h5p", `{"activity":"demo"}`}}
			if c.slug == "reading-worlds" {
				lessons = lessons[:1]
			}
			for i, l := range lessons {
				if e := tx.Exec("INSERT INTO lessons(module_id,title,description,position,type,config) VALUES (?,?,?,?,?,?::jsonb) ON CONFLICT(module_id,position) DO NOTHING", m.ID, l.title, l.desc, i+1, l.kind, l.conf).Error; e != nil {
					return e
				}
			}
		}
		return nil
	})
}
