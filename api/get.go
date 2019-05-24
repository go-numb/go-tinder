package api

import (
	"fmt"
	"sync"
	"time"

	tinder "github.com/DiSiqueira/TinderGo"
	"github.com/labstack/gommon/log"

	"golang.org/x/sync/errgroup"
)

type Tinders struct {
	mux sync.Mutex

	// 新しい情報があるかどうか
	New bool

	Records []Tinder

	Matches []tinder.Match
}
type Tinder struct {
	ID       string
	Name     string
	Birthday time.Time
	Age      int

	Distance int

	Bio string

	Filename  string
	Thumbnail string

	SNS SNS

	Posted    bool
	CreatedAt time.Time
}
type SNS struct {
	InstagramID string
}

func (p *Tinders) Get(token string) error {
	t := tinder.New()
	if err := t.Authenticate(token); err != nil {
		return err
	}

	var eg errgroup.Group

	eg.Go(func() error { // マッチする異性を探してくる
		/*
			geoにて座標指定も可能
			Defaultでは、登録時のGPS座標
		*/
		mat, err := t.Matches()
		if err != nil {
			return err
		}
		p.Matches = []tinder.Match{}
		for _, v := range mat {
			if len(v.Messages) == 0 {
				p.Matches = append(p.Matches, v)
			}
		}

		return nil
	})

	eg.Go(func() error {
		for {
			recs, err := t.RecsCore()
			if err != nil {
				return err
			}

			// fmt.Printf("%+v\n", recs[0])

			for _, v := range recs {
				var a Tinder
				a.ID = v.ID
				a.Name = v.Name
				a.Birthday = v.BirthDate
				// 年齢変換
				a.Age = int(time.Now().Sub(v.BirthDate).Hours() / 24 / 365)

				// 距離
				a.Distance = v.DistanceMi

				a.Bio = v.Bio
				a.Filename = v.Photos[0].URL
				a.Thumbnail = v.Photos[0].ProcessedFiles[1].URL
				if v.Instagram.Username != "" {
					a.SNS.InstagramID = v.Instagram.Username
				}
				a.CreatedAt = time.Now()

				// 年齢指定
				if a.Bio != "" && a.Age >= 20 && a.Age < 35 { // 自己紹介があればいいね＆Append
					res, err := t.Like(v)
					if err != nil {
						log.Error(err)
					}
					log.Infof("%+v", res)
					p.Records = append(p.Records, a)
				} else {
					res, err := t.Pass(v)
					if err != nil {
						log.Error(err)
					}
					log.Infof("%+v", res)
				}
			}

			if len(p.Records) > 3 {
				break
			}
			time.Sleep(5 * time.Second)
		}

		return nil
	})

	if err := eg.Wait(); err != nil {
		return err
	}

	// fmt.Printf("length: %d, %+v\n", len(p.Records), p.Records[0].Thumbnail)

	// 新規取得があれば
	p.New = true
	return nil
}

func (p *Tinders) AdjustLength(length int) {
	p.mux.Lock()
	defer p.mux.Unlock()

	l := len(p.Records)
	if l > length {
		p.Records = p.Records[l-length:]
	}
}

// Pirnt成形用
func (p *Tinders) String() string {
	p.mux.Lock()
	defer p.mux.Unlock()

	// 使い捨て用のmap
	m := make(map[string]struct{})
	// mainStructの入れ替え用
	temp := make([]Tinder, 0)

	for _, v := range p.Records {
		// mapの第二引数には値の存在真偽が入っているのでIDを使ってチェック
		if _, isThere := m[v.ID]; !isThere { // 入ってない場合tempに追加する
			m[v.ID] = struct{}{} // 追加したことをmapに教える
			temp = append(temp, v)
		}
	}

	p.Records = temp

	var str string
	if len(p.Matches) != 0 {
		str += fmt.Sprintf("マッチ後未返信案件が%d件あります\n", len(p.Matches))
	}
	for i, v := range p.Records {

		str += fmt.Sprintf("%d: %s(%d), %s\n%s\n\n", i+1, v.Name, v.Age, v.Thumbnail, v.Bio)
		p.Records[i].Posted = true

		if i > 4 {
			break
		}
	}

	return checkDiscordCharacters(str, 1998)
}

// Discord文字制限チェック
func checkDiscordCharacters(s string, l int) string {
	// the message contents (up to 2000 characters)
	if len([]rune(s)) > l {
		return string([]rune(s)[:l])
	}

	return s
}
