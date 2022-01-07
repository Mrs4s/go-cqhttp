package pages

import (
	"github.com/GoAdminGroup/go-admin/context"
	tmpl "github.com/GoAdminGroup/go-admin/template"
	"github.com/GoAdminGroup/go-admin/template/chartjs"
	"github.com/GoAdminGroup/go-admin/template/icon"
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/GoAdminGroup/themes/adminlte/components/chart_legend"
	"github.com/GoAdminGroup/themes/adminlte/components/description"
	"github.com/GoAdminGroup/themes/adminlte/components/infobox"
	"github.com/GoAdminGroup/themes/adminlte/components/productlist"
	"github.com/GoAdminGroup/themes/adminlte/components/progress_group"
	"github.com/GoAdminGroup/themes/adminlte/components/smallbox"
	"html/template"
)

func GetIndex(ctx *context.Context) (types.Panel, error) {
	components := tmpl.Default()
	colComp := components.Col()

	/**************************
	 * Info Box
	/**************************/

	infobox1 := infobox.New().
		SetText("CPU TRAFFIC").
		SetColor("aqua").
		SetNumber("100").
		SetIcon("ion-ios-gear-outline").
		GetContent()

	infobox2 := infobox.New().
		SetText("Likes").
		SetColor("red").
		SetNumber("1030.00<small>$</small>").
		SetIcon(icon.GooglePlus).
		GetContent()

	infobox3 := infobox.New().
		SetText("Sales").
		SetColor("green").
		SetNumber("760").
		SetIcon("ion-ios-cart-outline").
		GetContent()

	infobox4 := infobox.New().
		SetText("New Members").
		SetColor("yellow").
		SetNumber("2,349").
		SetIcon("ion-ios-people-outline"). // svg is ok
		GetContent()

	var size = types.SizeMD(3).SM(6).XS(12)
	infoboxCol1 := colComp.SetSize(size).SetContent(infobox1).GetContent()
	infoboxCol2 := colComp.SetSize(size).SetContent(infobox2).GetContent()
	infoboxCol3 := colComp.SetSize(size).SetContent(infobox3).GetContent()
	infoboxCol4 := colComp.SetSize(size).SetContent(infobox4).GetContent()
	row1 := components.Row().SetContent(infoboxCol1 + infoboxCol2 + infoboxCol3 + infoboxCol4).GetContent()

	/**************************
	 * Box
	/**************************/

	table := components.Table().SetType("table").SetInfoList([]map[string]types.InfoItem{
		{
			"Order ID":   {Content: "OR9842"},
			"Item":       {Content: "Call of Duty IV"},
			"Status":     {Content: "shipped"},
			"Popularity": {Content: "90%"},
		}, {
			"Order ID":   {Content: "OR9842"},
			"Item":       {Content: "Call of Duty IV"},
			"Status":     {Content: "shipped"},
			"Popularity": {Content: "90%"},
		}, {
			"Order ID":   {Content: "OR9842"},
			"Item":       {Content: "Call of Duty IV"},
			"Status":     {Content: "shipped"},
			"Popularity": {Content: "90%"},
		}, {
			"Order ID":   {Content: "OR9842"},
			"Item":       {Content: "Call of Duty IV"},
			"Status":     {Content: "shipped"},
			"Popularity": {Content: "90%"},
		},
	}).SetThead(types.Thead{
		{Head: "Order ID"},
		{Head: "Item"},
		{Head: "Status"},
		{Head: "Popularity"},
	}).GetContent()

	boxInfo := components.Box().
		WithHeadBorder().
		SetHeader("Latest Orders").
		SetHeadColor("#f7f7f7").
		SetBody(table).
		SetFooter(`<div class="clearfix"><a href="javascript:void(0)" class="btn btn-sm btn-info btn-flat pull-left">处理订单</a><a href="javascript:void(0)" class="btn btn-sm btn-default btn-flat pull-right">查看所有新订单</a> </div>`).
		GetContent()

	tableCol := colComp.SetSize(types.SizeMD(8)).SetContent(row1 + boxInfo).GetContent()

	/**************************
	 * Product List
	/**************************/

	productList := productlist.New().SetData([]map[string]string{
		{
			"img":         "//adminlte.io/themes/AdminLTE/dist/img/default-50x50.gif",
			"title":       "GoAdmin",
			"has_tabel":   "true",
			"labeltype":   "warning",
			"label":       "free",
			"description": `a framework help you build the dataviz system`,
		}, {
			"img":         "//adminlte.io/themes/AdminLTE/dist/img/default-50x50.gif",
			"title":       "GoAdmin",
			"has_tabel":   "true",
			"labeltype":   "warning",
			"label":       "free",
			"description": `a framework help you build the dataviz system`,
		}, {
			"img":         "//adminlte.io/themes/AdminLTE/dist/img/default-50x50.gif",
			"title":       "GoAdmin",
			"has_tabel":   "true",
			"labeltype":   "warning",
			"label":       "free",
			"description": `a framework help you build the dataviz system`,
		}, {
			"img":         "//adminlte.io/themes/AdminLTE/dist/img/default-50x50.gif",
			"title":       "GoAdmin",
			"has_tabel":   "true",
			"labeltype":   "warning",
			"label":       "free",
			"description": `a framework help you build the dataviz system`,
		},
	}).GetContent()

	boxWarning := components.Box().SetTheme("warning").WithHeadBorder().SetHeader("Recently Added Products").
		SetBody(productList).
		SetFooter(`<a href="javascript:void(0)" class="uppercase">View All Products</a>`).
		GetContent()

	newsCol := colComp.SetSize(types.SizeMD(4)).SetContent(boxWarning).GetContent()

	row5 := components.Row().SetContent(tableCol + newsCol).GetContent()

	/**************************
	 * Box
	/**************************/

	line := chartjs.Line()

	lineChart := line.
		SetID("salechart").
		SetHeight(180).
		SetTitle("Sales: 1 Jan, 2019 - 30 Jul, 2019").
		SetLabels([]string{"January", "February", "March", "April", "May", "June", "July"}).
		AddDataSet("Electronics").
		DSData([]float64{65, 59, 80, 81, 56, 55, 40}).
		DSFill(false).
		DSBorderColor("rgb(210, 214, 222)").
		DSLineTension(0.1).
		AddDataSet("Digital Goods").
		DSData([]float64{28, 48, 40, 19, 86, 27, 90}).
		DSFill(false).
		DSBorderColor("rgba(60,141,188,1)").
		DSLineTension(0.1).
		GetContent()

	title := `<p class="text-center"><strong>Goal Completion</strong></p>`
	progressGroup := progress_group.New().
		SetTitle("Add Products to Cart").
		SetColor("#76b2d4").
		SetDenominator(200).
		SetMolecular(160).
		SetPercent(80).
		GetContent()

	progressGroup1 := progress_group.New().
		SetTitle("Complete Purchase").
		SetColor("#f17c6e").
		SetDenominator(400).
		SetMolecular(310).
		SetPercent(80).
		GetContent()

	progressGroup2 := progress_group.New().
		SetTitle("Visit Premium Page").
		SetColor("#ace0ae").
		SetDenominator(800).
		SetMolecular(490).
		SetPercent(80).
		GetContent()

	progressGroup3 := progress_group.New().
		SetTitle("Send Inquiries").
		SetColor("#fdd698").
		SetDenominator(500).
		SetMolecular(250).
		SetPercent(50).
		GetContent()

	boxInternalCol1 := colComp.SetContent(lineChart).SetSize(types.SizeMD(8)).GetContent()
	boxInternalCol2 := colComp.
		SetContent(template.HTML(title) + progressGroup + progressGroup1 + progressGroup2 + progressGroup3).
		SetSize(types.SizeMD(4)).
		GetContent()

	boxInternalRow := components.Row().SetContent(boxInternalCol1 + boxInternalCol2).GetContent()

	description1 := description.New().
		SetPercent("17").
		SetNumber("¥140,100").
		SetTitle("TOTAL REVENUE").
		SetArrow("up").
		SetColor("green").
		SetBorder("right").
		GetContent()

	description2 := description.New().
		SetPercent("2").
		SetNumber("440,560").
		SetTitle("TOTAL REVENUE").
		SetArrow("down").
		SetColor("red").
		SetBorder("right").
		GetContent()

	description3 := description.New().
		SetPercent("12").
		SetNumber("¥140,050").
		SetTitle("TOTAL REVENUE").
		SetArrow("up").
		SetColor("green").
		SetBorder("right").
		GetContent()

	description4 := description.New().
		SetPercent("1").
		SetNumber("30943").
		SetTitle("TOTAL REVENUE").
		SetArrow("up").
		SetColor("green").
		GetContent()

	size2 := types.SizeSM(3).XS(6)
	boxInternalCol3 := colComp.SetContent(description1).SetSize(size2).GetContent()
	boxInternalCol4 := colComp.SetContent(description2).SetSize(size2).GetContent()
	boxInternalCol5 := colComp.SetContent(description3).SetSize(size2).GetContent()
	boxInternalCol6 := colComp.SetContent(description4).SetSize(size2).GetContent()

	boxInternalRow2 := components.Row().SetContent(boxInternalCol3 + boxInternalCol4 + boxInternalCol5 + boxInternalCol6).GetContent()

	box := components.Box().WithHeadBorder().SetHeader("Monthly Recap Report").
		SetBody(boxInternalRow).
		SetFooter(boxInternalRow2).
		GetContent()

	boxcol := colComp.SetContent(box).SetSize(types.SizeMD(12)).GetContent()
	row2 := components.Row().SetContent(boxcol).GetContent()

	/**************************
	 * Small Box
	/**************************/

	smallbox1 := smallbox.New().SetColor("blue").SetIcon("ion-ios-gear-outline").SetUrl("/").SetTitle("new users").SetValue("345￥").GetContent()
	smallbox2 := smallbox.New().SetColor("yellow").SetIcon("ion-ios-cart-outline").SetUrl("/").SetTitle("new users").SetValue("80%").GetContent()
	smallbox3 := smallbox.New().SetColor("red").SetIcon("fa-user").SetUrl("/").SetTitle("new users").SetValue("645￥").GetContent()
	smallbox4 := smallbox.New().SetColor("green").SetIcon("ion-ios-cart-outline").SetUrl("/").SetTitle("new users").SetValue("889￥").GetContent()

	col1 := colComp.SetSize(size).SetContent(smallbox1).GetContent()
	col2 := colComp.SetSize(size).SetContent(smallbox2).GetContent()
	col3 := colComp.SetSize(size).SetContent(smallbox3).GetContent()
	col4 := colComp.SetSize(size).SetContent(smallbox4).GetContent()

	row3 := components.Row().SetContent(col1 + col2 + col3 + col4).GetContent()

	/**************************
	 * Pie Chart
	/**************************/

	pie := chartjs.Pie().
		SetHeight(170).
		SetLabels([]string{"Navigator", "Opera", "Safari", "FireFox", "IE", "Chrome"}).
		SetID("pieChart").
		AddDataSet("Chrome").
		DSData([]float64{100, 300, 600, 400, 500, 700}).
		DSBackgroundColor([]chartjs.Color{
			"rgb(255, 205, 86)", "rgb(54, 162, 235)", "rgb(255, 99, 132)", "rgb(255, 205, 86)", "rgb(54, 162, 235)", "rgb(255, 99, 132)",
		}).
		GetContent()

	legend := chart_legend.New().SetData([]map[string]string{
		{
			"label": " Chrome",
			"color": "red",
		}, {
			"label": " IE",
			"color": "Green",
		}, {
			"label": " FireFox",
			"color": "yellow",
		}, {
			"label": " Sarafri",
			"color": "blue",
		}, {
			"label": " Opera",
			"color": "light-blue",
		}, {
			"label": " Navigator",
			"color": "gray",
		},
	}).GetContent()

	boxDanger := components.Box().SetTheme("danger").WithHeadBorder().SetHeader("Browser Usage").
		SetBody(components.Row().
			SetContent(colComp.SetSize(types.SizeMD(8)).
				SetContent(pie).
				GetContent() + colComp.SetSize(types.SizeMD(4)).
				SetContent(legend).
				GetContent()).GetContent()).
		SetFooter(`<p class="text-center"><a href="javascript:void(0)" class="uppercase">View All Users</a></p>`).
		GetContent()

	tabs := components.Tabs().SetData([]map[string]template.HTML{
		{
			"title": "tabs1",
			"content": template.HTML(`<b>How to use:</b>

<p>Exactly like the original bootstrap tabs except you should use
the custom wrapper <code>.nav-tabs-custom</code> to achieve this style.</p>
A wonderful serenity has taken possession of my entire soul,
like these sweet mornings of spring which I enjoy with my whole heart.
I am alone, and feel the charm of existence in this spot,
which was created for the bliss of souls like mine. I am so happy,
my dear friend, so absorbed in the exquisite sense of mere tranquil existence,
that I neglect my talents. I should be incapable of drawing a single stroke
at the present moment; and yet I feel that I never was a greater artist than now.`),
		}, {
			"title": "tabs2",
			"content": template.HTML(`
The European languages are members of the same family. Their separate existence is a myth.
For science, music, sport, etc, Europe uses the same vocabulary. The languages only differ
in their grammar, their pronunciation and their most common words. Everyone realizes why a
new common language would be desirable: one could refuse to pay expensive translators. To
achieve this, it would be necessary to have uniform grammar, pronunciation and more common
words. If several languages coalesce, the grammar of the resulting language is more simple
and regular than that of the individual languages.
`),
		}, {
			"title": "tabs3",
			"content": template.HTML(`
Lorem Ipsum is simply dummy text of the printing and typesetting industry.
Lorem Ipsum has been the industry's standard dummy text ever since the 1500s,
when an unknown printer took a galley of type and scrambled it to make a type specimen book.
It has survived not only five centuries, but also the leap into electronic typesetting,
remaining essentially unchanged. It was popularised in the 1960s with the release of Letraset
sheets containing Lorem Ipsum passages, and more recently with desktop publishing software
like Aldus PageMaker including versions of Lorem Ipsum.
`),
		},
	}).GetContent()

	buttonTest := `<button type="button" class="btn btn-primary" data-toggle="modal" data-target="#exampleModal" data-whatever="@mdo">Open modal for @mdo</button>`
	popupForm := `<form>
<div class="form-group">
<label for="recipient-name" class="col-form-label">Recipient:</label>
<input type="text" class="form-control" id="recipient-name">
</div>
<div class="form-group">
<label for="message-text" class="col-form-label">Message:</label>
<textarea class="form-control" id="message-text"></textarea>
</div>
</form>`
	popup := components.Popup().SetID("exampleModal").
		SetFooter("Save Change").
		SetTitle("this is a popup").
		SetBody(template.HTML(popupForm)).
		GetContent()

	col5 := colComp.SetSize(types.SizeMD(8)).SetContent(tabs + template.HTML(buttonTest)).GetContent()
	col6 := colComp.SetSize(types.SizeMD(4)).SetContent(boxDanger + popup).GetContent()

	row4 := components.Row().SetContent(col5 + col6).GetContent()

	return types.Panel{
		Content:     row3 + row2 + row5 + row4,
		Title:       "Dashboard",
		Description: "dashboard example",
	}, nil
}
