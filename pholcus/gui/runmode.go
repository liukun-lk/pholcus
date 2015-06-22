package gui

import (
	"github.com/henrylee2cn/pholcus/config"
	. "github.com/lxn/walk/declarative"
	"log"
)

func runmodeWindow() {
	if err := (MainWindow{
		AssignTo: &mw,
		DataBinder: DataBinder{
			AssignTo:       &db,
			DataSource:     Input,
			ErrorPresenter: ErrorPresenterRef{&ep},
		},
		Title:   config.APP_NAME,
		MinSize: Size{450, 350},
		Layout:  VBox{ /*MarginsZero: true*/ },
		Children: []Widget{

			RadioButtonGroupBox{
				AssignTo: &mode,
				Title:    "*运行模式",
				Layout:   HBox{},
				MinSize:  Size{0, 70},

				DataMember: "RunMode",
				Buttons: []RadioButton{
					{Text: config.GuiOpt.RunMode[0].Key, Value: config.GuiOpt.RunMode[0].Int},
					{Text: config.GuiOpt.RunMode[1].Key, Value: config.GuiOpt.RunMode[1].Int},
					{Text: config.GuiOpt.RunMode[2].Key, Value: config.GuiOpt.RunMode[2].Int},
				},
			},

			VSplitter{
				AssignTo: &host,
				MaxSize:  Size{0, 120},
				Children: []Widget{
					VSplitter{
						Children: []Widget{
							Label{
								Text: "分布式端口：（单机模式不填）",
							},
							NumberEdit{
								Value:    Bind("Port"),
								Suffix:   "",
								Decimals: 0,
							},
						},
					},

					VSplitter{
						Children: []Widget{
							Label{
								Text: "主节点 URL：（客户端模式必填）",
							},
							LineEdit{
								Text: Bind("Master"),
							},
						},
					},
				},
			},

			PushButton{
				Text:     "确认开始",
				MinSize:  Size{0, 30},
				AssignTo: &toggleSpecialModePB,
				OnClicked: func() {
					if err := db.Submit(); err != nil {
						log.Println(err)
						return
					}

					// 配置运行模式
					writeConf1()

					switch Input.RunMode {
					case config.OFFLINE:
						offlineWindow()

					case config.SERVER:
						serverWindow()

					case config.CLIENT:
						clientWindow()
					}
				},
			},
		},
	}.Create()); err != nil {
		log.Fatal(err)
	}
	// 运行窗体程序
	mw.Run()
}
