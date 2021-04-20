package CVRidcard

import (
    "bytes"
    "golang.org/x/text/encoding/simplifiedchinese"
    "golang.org/x/text/transform"
    "io/ioutil"
    "os"
    "os/user"
    "syscall"
    "runtime"
    "bufio"
    "encoding/base64"
    "unsafe"

)

const (
    CVR_PORT  = 1001   /*端口号*/
    CVR_ACTIVE = 4     /*临时目录中保存哪些文件*/   
)

var (
    Path =  dllPate()
    DllPath            = Path+"\\Termb.dll"
    CVRDll              = syscall.NewLazyDLL(DllPath)

    CVR_InitComm        = CVRDll.NewProc("CVR_InitComm")         //初始化连接   int CVR_InitComm(int Port) 
    CVR_Authenticate    = CVRDll.NewProc("CVR_Authenticate")     //卡认证  int CVR_Authenticate()
    CVR_Read_Content    = CVRDll.NewProc("CVR_Read_Content")     //读卡操作   int CVR_Read_Content(int active)
    CVR_Read_FPContent  = CVRDll.NewProc("CVR_Read_FPContent")   //读卡操作，含指纹  int CVR_Read_FPContent(int active)
    CVR_CloseComm       = CVRDll.NewProc("CVR_CloseComm")        //关闭连接   int CVR_InitComm(int Port) 

    // 以下为可选API函数,多字节版本
    PeopleName       = CVRDll.NewProc("GetPeopleName")        // 姓名信息 不超过30字节  int  GetPeopleName(char *strTmp, int *strLen)
    PeopleSex        = CVRDll.NewProc("GetPeopleSex")         // 性别信息 不超过2个字节  int  GetPeopleSex(char *strTmp, int *strLen)
    PeopleNation     = CVRDll.NewProc("GetPeopleNation")      // 民族信息      int  GetPeopleNation(char *strTmp, int *strLen)
    PeopleBirthday   = CVRDll.NewProc("GetPeopleBirthday")    // 出生日期      int  GetPeopleBirthday(char *strTmp, int *strLen)
    PeopleIDCode     = CVRDll.NewProc("GetPeopleIDCode")      // 身份证号信息  int  GetPeopleIDCode(char *strTmp, int *strLen)
    Department       = CVRDll.NewProc("GetDepartment")        // 发证机关信息  int  GetDepartment(char *strTmp, int *strLen)
    StartDate        = CVRDll.NewProc("GetStartDate")         // 有效开始日期
    EndDate          = CVRDll.NewProc("GetEndDate")           // 有效截止日期
    NationCode       = CVRDll.NewProc("GetNationCode")        // 居民民族代码

    FPDate           = CVRDll.NewProc("GetFPDate")            // 指纹数据
    PeopleAddress    = CVRDll.NewProc("GetPeopleAddress")     // 地址信息
    PassCheckID      = CVRDll.NewProc("GetPassCheckID")       // 通行证号码
    IssuesNum        = CVRDll.NewProc("GetIssuesNum")         // 签发次数  int *IssuesNum
    NewAppMsg        = CVRDll.NewProc("GetNewAppMsg")         // 获取追加地址
    BMPData          = CVRDll.NewProc("GetBMPData")           // 得到头像照片bmp数据，不超过38862字节
    base64BMPData    = CVRDll.NewProc("Getbase64BMPData")     // 头像照片base64编码数据，不超过38862*2字节  int Getbase64BMPData (unsigned char *pData, int * pLen)     
    base64JpgData    = CVRDll.NewProc("Getbase64JpgData")     // 头像照片jpg数据
    SAMID    = CVRDll.NewProc("CVR_GetSAMID")     // 安全模块号  int  CVR_GetSAMID(char *SAMID)   
)

func Initialize() uintptr{

    e1, _, _ := CVR_InitComm.Call(uintptr(CVR_PORT))
     if e1 != 0 {
         e2, _, _ := CVR_Authenticate.Call()

         if e2 != 0 {
              e3, _ ,_ := CVR_Read_Content.Call(uintptr(CVR_ACTIVE))
              return e3 
         }
        return e2 
     }

    return e1

}

type IdcardMap struct {
    Name string
    Sex  string
    Nation string
    Birthday string
    PeopleAddress string
    IDCode string
    Department string
    StartEndDate string
    JpgData string
}



func GetIdCdrdInfo(jpg bool)(IdcardMap, error){
   
    s := IdcardMap{}
    sourcestring := ""
    jpgData := ""

    err1 := Initialize()

    if err1 != 1 {
        CVR_CloseComm .Call()    
        return s ,nil     
    }
            if runtime.GOOS == "windows"{
                CVR_CloseComm .Call()
                //fmt.Println(runtime.GOOS)
                u, _ := user.Current()
                wz_path := u.HomeDir+"\\AppData\\Local\\Temp\\chinaidcard\\wz.txt"
                //fmt.Println("wz_path:",wz_path) 
                e, err := ReadWZ(wz_path)
                //fmt.Println("ReadWZ:",e[1])
                
                if err != nil {
                    return s, err           
                }
               if jpg {
                    jpgPath := u.HomeDir+"\\AppData\\Local\\Temp\\chinaidcard\\xp.jpg"
                    ff, _ := os.Open(jpgPath)
                    defer ff.Close()
                    sourcebuffer := make([]byte, 50000)
                    n, _ := ff.Read(sourcebuffer)
                    //base64压缩
                    sourcestring = base64.StdEncoding.EncodeToString(sourcebuffer[:n])
               }


                s = IdcardMap{e[1],e[2],e[3],e[4],e[5],e[6],e[7],e[8],sourcestring}
                return s, nil
            }
        //linux
        
            s.Name, _ = GetPeopleName()
            s.Sex, _  =GetPeopleSex()
            s.Nation, _ =GetPeopleNation()
            s.Birthday, _ =GetPeopleBirthday()
            s.PeopleAddress, _ = GetPeopleAddress()
            s.IDCode, _   =GetPeopleIDCode()
            s.Department, _  =GetDepartment()
            sd, _ := GetStartDate()
            ed, _ := GetEndDate()
            s.StartEndDate = sd+"-"+ed
            if jpg {
               jpgData,_ = Getbase64JpgData()
            }
            s.JpgData = jpgData

        
        CVR_CloseComm .Call()
        return s ,nil
}


//编码转换
func GbkToUtf8(s []byte) ([]byte, error) {
    reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewDecoder())
    d, e := ioutil.ReadAll(reader)
    if e != nil {
        return nil, e
    }
    return d, nil
}

func Utf8ToGbk(s []byte) ([]byte, error) {
    reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewEncoder())
    d, e := ioutil.ReadAll(reader)
    if e != nil {
        return nil, e
    }
    return d, nil
}

func dllPate() string {
    dir,_ := os.Getwd()
    if runtime.GOARCH == "386" {
        return dir+"\\32bit"
    }
    return dir+"\\64bit"
}

func ReadWZ(filename string) (map[int]string, error){
    // Read UTF-8 from a GBK encoded file.
    idcardInfo := make(map[int]string)
    f, err := os.Open(filename)
    if err != nil {
        return idcardInfo,err
    }
    r := transform.NewReader(f, simplifiedchinese.GBK.NewDecoder())
    sc := bufio.NewScanner(r)
    count := 1
    for sc.Scan() {

        idcardInfo[count] = string(sc.Bytes())
         count++
        
    }
    if err = sc.Err(); err != nil {
        return idcardInfo, err
    }

    if err = f.Close(); err != nil {
        return idcardInfo,err
    }

    return idcardInfo, nil
    
}


 func GetPeopleName()(string, string){
    var a = make([]byte, 30)
    var b int = len(a)
    err := "err"    
        r,_,_ := PeopleName.Call(uintptr(unsafe.Pointer(&a[0])),uintptr(unsafe.Pointer(&b)))         
        if r == 1 {
            a, _ = GbkToUtf8(a)
            err = ""
        }  
  return string(a), err
}

 func GetPeopleSex()(string, string){
    var a = make([]byte, 3)
    var b int = len(a)
    err := "err"    
        r,_,_ := PeopleSex.Call(uintptr(unsafe.Pointer(&a[0])),uintptr(unsafe.Pointer(&b)))         
        if r == 1 {
            a, _ = GbkToUtf8(a)
            err = ""
        }  
  return string(a), err
}


 func GetPeopleNation()(string, string){
    var a = make([]byte, 30)
    var b int = len(a)
    err := "err"    
        r,_,_ := PeopleNation.Call(uintptr(unsafe.Pointer(&a[0])),uintptr(unsafe.Pointer(&b)))         
        if r == 1 {
            a, _ = GbkToUtf8(a)
            err = ""
        }  
  return string(a), err
}

 func GetPeopleBirthday()(string, string){
    var a = make([]byte, 10)
    var b int = len(a)
    err := "err"    
        r,_,_ := PeopleBirthday.Call(uintptr(unsafe.Pointer(&a[0])),uintptr(unsafe.Pointer(&b)))         
        if r == 1 {
            a, _ = GbkToUtf8(a)
            err = ""
        }  
  return string(a), err
}

 func GetPeopleAddress()(string, string){
    var a = make([]byte, 60)
    var b int = len(a)
    err := "err"    
        r,_,_ := PeopleAddress.Call(uintptr(unsafe.Pointer(&a[0])),uintptr(unsafe.Pointer(&b)))         
        if r == 1 {
            a, _ = GbkToUtf8(a)
            err = ""
        }  
  return string(a), err
}

 func GetPeopleIDCode()(string, string){
    var a = make([]byte, 20)
    var b int = len(a)
    err := "err"    
        r,_,_ := PeopleIDCode.Call(uintptr(unsafe.Pointer(&a[0])),uintptr(unsafe.Pointer(&b)))         
        if r == 1 {
            a, _ = GbkToUtf8(a)
            err = ""
        }  
  return string(a), err
}

 func GetDepartment()(string, string){
    var a = make([]byte, 30)
    var b int = len(a)
    err := "err"    
        r,_,_ := Department.Call(uintptr(unsafe.Pointer(&a[0])),uintptr(unsafe.Pointer(&b)))         
        if r == 1 {
            a, _ = GbkToUtf8(a)
            err = ""
        }  
  return string(a), err
}

 func GetStartDate()(string, string){
    var a = make([]byte, 10)
    var b int = len(a)
    err := "err"    
        r,_,_ := StartDate.Call(uintptr(unsafe.Pointer(&a[0])),uintptr(unsafe.Pointer(&b)))         
        if r == 1 {
            a, _ = GbkToUtf8(a)
            err = ""
        }  
  return string(a), err
}



 func GetEndDate()(string, string){
    var a = make([]byte, 10)
    var b int = len(a)
    err := "err"    
        r,_,_ := EndDate.Call(uintptr(unsafe.Pointer(&a[0])),uintptr(unsafe.Pointer(&b)))         
        if r == 1 {
            a, _ = GbkToUtf8(a)
            err = ""
        }  
  return string(a), err
}

 func Getbase64JpgData()(string, string){
    var a = make([]byte, 40000)
    var b int = len(a)
    err := "err"    
        r,_,_ := base64JpgData.Call(uintptr(unsafe.Pointer(&a[0])),uintptr(unsafe.Pointer(&b)))         
        if r == 1 {
            a, _ = GbkToUtf8(a)
            err = ""
        }  
  return string(a), err
}

