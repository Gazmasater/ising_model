<script setup lang="ts">
useHead({
  title: "Ремонт газовых котлов в Липецке",
  meta: [
    {
      name: "description",
      content:
        "Ремонт газовых котлов в Липецке. Выезд в день обращения, бесплатная диагностика при ремонте, гарантия на работы. Звоните: +7 (962) 352-70-02.",
    },
  ],
});
</script>

<template>
  <main class="page">
    <section class="hero">
      <img
        src="/img/header-boiler.png"
        alt="Ремонт газовых котлов в Липецке"
        class="hero-image"
      />
    </section>

    <section class="content">
      <div class="container">
        <h1>Ремонт газовых котлов в Липецке</h1>

        <p class="lead">
          Если котёл перестал запускаться, выдаёт ошибку, шумит, гаснет или не
          греет воду — не откладывайте ремонт. Чем раньше мастер проведёт
          диагностику, тем выше шанс устранить неисправность быстро и без
          дорогостоящей замены узлов.
        </p>

        <p>
          Мы выполняем ремонт газовых котлов <strong>всех популярных марок</strong>
          в Липецке и области: настенных, напольных, двухконтурных и
          одноконтурных моделей. Работаем с неисправностями розжига, платы
          управления, датчиков, вентилятора, теплообменника, насоса,
          трёхходового клапана и системы дымоудаления.
        </p>

        <p>
          Мастер приедет <strong>в день обращения</strong>, проведёт проверку,
          найдёт причину поломки и предложит оптимальное решение. Мы объясняем,
          что именно вышло из строя, какие есть варианты ремонта и сколько это
          будет стоить до начала работ. Без навязанных услуг и лишних замен.
        </p>

        <div class="benefits">
          <div class="benefit">
            <h2>Почему обращаются к нам</h2>
            <ul>
              <li>Выезд мастера в день обращения</li>
              <li>Бесплатная диагностика при ремонте</li>
              <li>Ремонт большинства неисправностей за 1 визит</li>
              <li>Гарантия на выполненные работы</li>
              <li>Честные цены без скрытых доплат</li>
            </ul>
          </div>

          <div class="benefit">
            <h2>Когда нужен срочный ремонт</h2>
            <ul>
              <li>Котёл не включается или сразу отключается</li>
              <li>Нет отопления или горячей воды</li>
              <li>Появилась ошибка на дисплее</li>
              <li>Упал напор или температура нестабильна</li>
              <li>Слышны хлопки, стуки, гул, посторонний шум</li>
              <li>Пахнет газом — сразу перекройте газ и вызовите специалиста</li>
            </ul>
          </div>
        </div>

        <div class="cta">
          <p class="cta-title">Нужен мастер по ремонту газового котла?</p>
          <p class="cta-text">
            Позвоните прямо сейчас — проконсультируем по симптомам поломки,
            сориентируем по выезду и запишем мастера на удобное время.
          </p>
          <a class="phone" href="tel:+79623527002">+7 (962) 352-70-02</a>
        </div>
      </div>
    </section>
  </main>
</template>

<style scoped>
.page {
  background: #f5f7fb;
  color: #1d2a3a;
  min-height: 100vh;
}

.hero {
  width: 100%;
  background: #0a1a39;
}

.hero-image {
  display: block;
  width: 100%;
  max-height: 720px;
  object-fit: cover;
}

.content {
  padding: 48px 16px 72px;
}

.container {
  max-width: 1100px;
  margin: 0 auto;
}

h1 {
  margin: 0 0 24px;
  font-size: 42px;
  line-height: 1.15;
  color: #0f2247;
}

.lead {
  font-size: 22px;
  line-height: 1.6;
  color: #243b63;
  margin-bottom: 24px;
}

p {
  font-size: 18px;
  line-height: 1.75;
  margin: 0 0 20px;
}

.benefits {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 24px;
  margin: 40px 0;
}

.benefit {
  background: #ffffff;
  border-radius: 20px;
  padding: 28px;
  box-shadow: 0 10px 30px rgba(15, 34, 71, 0.08);
}

.benefit h2 {
  margin: 0 0 18px;
  font-size: 26px;
  color: #0f2247;
}

.benefit ul {
  margin: 0;
  padding-left: 22px;
}

.benefit li {
  margin-bottom: 12px;
  font-size: 17px;
  line-height: 1.6;
}

.cta {
  background: linear-gradient(135deg, #0f2247, #1e4fa8);
  color: #fff;
  border-radius: 24px;
  padding: 32px;
  text-align: center;
  margin-top: 24px;
}

.cta-title {
  font-size: 30px;
  font-weight: 700;
  margin-bottom: 12px;
  color: #fff;
}

.cta-text {
  font-size: 18px;
  line-height: 1.7;
  margin-bottom: 20px;
  color: rgba(255, 255, 255, 0.92);
}

.phone {
  display: inline-block;
  text-decoration: none;
  background: #ffd200;
  color: #111;
  font-weight: 800;
  font-size: 32px;
  padding: 16px 28px;
  border-radius: 18px;
  transition: transform 0.2s ease;
}

.phone:hover {
  transform: translateY(-2px);
}

@media (max-width: 900px) {
  h1 {
    font-size: 32px;
  }

  .lead {
    font-size: 19px;
  }

  .benefits {
    grid-template-columns: 1fr;
  }

  .cta-title {
    font-size: 24px;
  }

  .phone {
    font-size: 24px;
    width: 100%;
    box-sizing: border-box;
  }
}

@media (max-width: 600px) {
  .content {
    padding: 32px 14px 56px;
  }

  h1 {
    font-size: 28px;
  }

  p,
  .benefit li {
    font-size: 16px;
  }

  .benefit {
    padding: 22px;
  }

  .cta {
    padding: 24px 18px;
  }
}
</style>




app/public/img/header-boiler.png
